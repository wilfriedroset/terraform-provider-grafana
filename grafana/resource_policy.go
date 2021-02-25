package grafana

import (
	"errors"
	"fmt"
	"log"
	"strings"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform/helper/schema"
)

func ResourcePolicy() *schema.Resource {
	return &schema.Resource{
		Create: CreatePolicy,
		Update: UpdatePolicy,
		Read:   ReadPolicy,
		Delete: DeletePolicy,
		Exists: ExistsPolicy,
		Importer: &schema.ResourceImporter{
			State: ImportPolicy,
		},
		Schema: map[string]*schema.Schema{
			"org_id": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"permissions": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeMap,
				},
			},
		},
	}
}

func CreatePolicy(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	perms := permissions(d)
	orgId, _ := d.Get("org_id").(int)
	policy := gapi.Policy{
		OrgID:       int64(orgId),
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		Permissions: perms,
	}
	uid, err := client.NewPolicy(policy)
	if err != nil {
		return err
	}
	d.SetId(uid)
	return nil
}

func permissions(d *schema.ResourceData) []gapi.Permission {
	rp := d.Get("permissions").([]interface{})

	perms := make([]gapi.Permission, 0)
	for _, p := range rp {
		ps := p.(map[string]interface{})
		perms = append(perms, gapi.Permission{
			Permission: ps["Permission"].(string),
			Scope:      ps["Scope"].(string),
		})
	}

	return perms
}

func ReadPolicy(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)
	uid := d.Id()
	p, err := client.GetPolicy(uid)

	if err != nil && strings.HasPrefix(err.Error(), "status: 404") {
		log.Printf("[WARN] removing policy %s from state because it no longer exists in grafana", d.Id())
		d.SetId("")
		return nil
	}
	if err != nil {
		return err
	}
	err = d.Set("name", p.Name)
	if err != nil {
		return err
	}
	err = d.Set("description", p.Description)
	if err != nil {
		return err
	}
	perms := make([]interface{}, 0)
	for _, perm := range p.Permissions {
		permMap := make(map[string]interface{})
		permMap["Permission"] = perm.Permission
		permMap["Scope"] = perm.Scope
		perms = append(perms, permMap)
	}
	err = d.Set("permissions", perms)
	if err != nil {
		return err
	}
	err = d.Set("org_id", p.OrgID)
	if err != nil {
		return err
	}
	return nil
}

func UpdatePolicy(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)
	uid := d.Id()

	if d.HasChange("name") || d.HasChange("description") || d.HasChange("permissions") || d.HasChange("org_id") {
		name := d.Get("name").(string)
		description := d.Get("description").(string)
		orgId, _ := d.Get("org_id").(int)
		perms := permissions(d)
		policy := gapi.Policy{
			OrgID:       int64(orgId),
			Name:        name,
			Description: description,
			Permissions: perms,
		}
		err := client.UpdatePolicy(uid, policy)
		if err != nil {
			return nil
		}
	}

	return nil
}

func DeletePolicy(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)
	uid := d.Id()
	err := client.DeletePolicy(uid)
	return err
}

func ExistsPolicy(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*gapi.Client)
	uid := d.Id()
	_, err := client.GetPolicy(uid)

	if err != nil && strings.HasPrefix(err.Error(), "status: 404") {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, err
}

func ImportPolicy(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	exists, err := ExistsPolicy(d, meta)
	if err != nil || !exists {
		return nil, errors.New(fmt.Sprintf("Error: Unable to import Grafana Policy: %s.", err))
	}
	err = ReadPolicy(d, meta)
	if err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}
