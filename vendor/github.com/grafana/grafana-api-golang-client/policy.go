package gapi

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type Policy struct {
	OrgID       int64        `json:"orgId"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Permissions []Permission `json:"permissions,omitempty"`
}

type Permission struct {
	Permission string `json:"permission"`
	Scope      string `json:"scope"`
}

func (c *Client) NewPolicy(policy Policy) (string, error) {
	data, err := json.Marshal(policy)
	if err != nil {
		return "", err
	}

	created := struct {
		UID string `json:"uid"`
	}{}

	err = c.request("POST", "/api/access-control/policies", nil, bytes.NewBuffer(data), &created)
	if err != nil {
		return "", err
	}

	return created.UID, err
}

func (c *Client) UpdatePolicy(uid string, policy Policy) error {
	data, err := json.Marshal(policy)
	if err != nil {
		return err
	}

	err = c.request("PUT", fmt.Sprintf("/api/access-control/policies/%s", uid), nil, bytes.NewBuffer(data), nil)

	return err
}

func (c *Client) DeletePolicy(uid string) error {
	return c.request("DELETE", fmt.Sprintf("/api/access-control/policies/%s", uid), nil, nil, nil)
}
