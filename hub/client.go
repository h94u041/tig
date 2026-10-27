package hub

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/h94u041/tig"
)

// PushResult is how a push went.
type PushResult struct {
	Publication Publication
	UpToDate    bool     // the hub already had these documents and this chain
	Sent        int      // objects sent
	Unsnapped   []string // documents changed since the latest snapshot, so not pushed as they are now
}

// Push sends the project's documents and chain to a hub. project is the
// project's address there, http://host/owner/name.
func Push(ctx context.Context, r *tig.Repo, project, token, actor, note string) (PushResult, error) {
	api, err := apiURL(project)
	if err != nil {
		return PushResult{}, err
	}
	c := &client{api: api, token: token}
	var res PushResult
	if res.Unsnapped, err = unsnapped(ctx, r); err != nil {
		return res, err
	}

	var info ProjectInfo
	if err := c.call(ctx, "GET", "", nil, &info); err != nil {
		return res, err
	}
	old := ""
	if info.Head != nil {
		old = info.Head.ID
	}
	pub, objects, err := Build(ctx, r, old, actor, note)
	if err != nil {
		return res, err
	}
	res.Publication = pub
	if info.Head != nil && info.Head.Docs == pub.Docs && info.Head.Chain == pub.Chain {
		res.Publication, res.UpToDate = *info.Head, true
		return res, nil
	}

	want := missingRequest{}
	for h := range objects {
		want.Objects = append(want.Objects, h)
	}
	slices.Sort(want.Objects)
	var missing missingResponse
	if err := c.call(ctx, "POST", "/missing", want, &missing); err != nil {
		return res, err
	}
	push := pushRequest{Old: old, New: pub.ID, Objects: map[string][]byte{}}
	for _, h := range missing.Missing {
		if data, ok := objects[h]; ok {
			push.Objects[h] = data
		}
	}
	res.Sent = len(push.Objects)
	return res, c.call(ctx, "POST", "/push", push, nil)
}

// unsnapped are the documents of the chain that changed since the latest
// snapshot.
func unsnapped(ctx context.Context, r *tig.Repo) ([]string, error) {
	head, err := r.Head(ctx)
	if err != nil || head.ID == "" {
		return nil, err
	}
	changes, err := r.Changed(ctx, head.Tree)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, ch := range changes {
		if tig.IsMarkdown(ch.Path) {
			out = append(out, ch.Path)
		}
	}
	return out, nil
}

// apiURL is the API address of a project from its address on the hub.
func apiURL(project string) (string, error) {
	u, err := url.Parse(strings.TrimSuffix(project, "/"))
	if err != nil {
		return "", err
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || len(parts) < 2 {
		return "", fmt.Errorf("hub: %q is not a project's address on a hub (http://host/owner/name)", project)
	}
	owner, name := parts[len(parts)-2], parts[len(parts)-1]
	if !validName(owner) || !validName(name) {
		return "", fmt.Errorf("hub: %q is not a valid owner and project name", owner+"/"+name)
	}
	u.Path = strings.Join(parts[:len(parts)-2], "/")
	return strings.TrimSuffix(u.String(), "/") + "/api/v1/projects/" + owner + "/" + name, nil
}

type client struct {
	api, token string
}

func (c *client) call(ctx context.Context, method, suffix string, body, out any) error {
	var rd io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.api+suffix, rd)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxPush))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		var e apiError
		if json.Unmarshal(data, &e) == nil && e.Error != "" {
			return fmt.Errorf("hub: %s", e.Error)
		}
		return fmt.Errorf("hub: %s %s: %s", method, c.api+suffix, resp.Status)
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(data, out)
}
