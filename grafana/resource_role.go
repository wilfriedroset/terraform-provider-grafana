package grafana

import (
	"errors"
	"fmt"
	"log"
	"strings"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform/helper/schema"
)

func ResourceRole() *schema.Resource {
	return &schema.Resource{
		Create: CreateRole,
		Update: UpdateRole,
		Read:   ReadRole,
		Delete: DeleteRole,
		Exists: ExistsRole,
		Importer: &schema.ResourceImporter{
			State: ImportRole,
		},
		Schema: map[string]*schema.Schema{
			"org_id": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"uid": {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				ForceNew: true,
			},
			"version": {
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
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"action": {
							Type: schema.TypeString,
							Required: true,
						},
						"scope": {
							Type: schema.TypeString,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func CreateRole(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	perms := permissions(d)
	orgId, _ := d.Get("org_id").(int)
	version, _ := d.Get("version").(int)
	role := gapi.Role{
		OrgID:       int64(orgId),
		UID:         d.Get("uid").(string),
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		Version:     int64(version),
		Permissions: perms,
	}
	r, err := client.NewRole(role)
	if err != nil {
		return err
	}
	err = d.Set("uid", r.UID)
	if err != nil {
		return err
	}
	d.SetId(r.UID)
	return nil
}

func permissions(d *schema.ResourceData) []gapi.Permission {
	v, ok := d.GetOk("permissions")
	if !ok {
		return nil
	}

	perms := make([]gapi.Permission, 0)
	for _, permission := range v.(*schema.Set).List() {
		permission := permission.(map[string]interface{})
		perms = append(perms, gapi.Permission{
			Action: permission["action"].(string),
			Scope: permission["scope"].(string),
		})
	}

	return perms
}

func ReadRole(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)
	uid := d.Id()
	r, err := client.GetRole(uid)

	if err != nil && strings.HasPrefix(err.Error(), "status: 404") {
		log.Printf("[WARN] removing role %s from state because it no longer exists in grafana", d.Id())
		d.SetId("")
		return nil
	}
	if err != nil {
		return err
	}
	err = d.Set("name", r.Name)
	if err != nil {
		return err
	}
	err = d.Set("uid", r.UID)
	if err != nil {
		return err
	}
	err = d.Set("description", r.Description)
	if err != nil {
		return err
	}
	perms := make([]interface{}, 0)
	for _, perm := range r.Permissions {
		permMap := make(map[string]interface{})
		permMap["action"] = perm.Action
		permMap["scope"] = perm.Scope
		perms = append(perms, permMap)
	}
	err = d.Set("permissions", perms)
	if err != nil {
		return err
	}
	err = d.Set("org_id", r.OrgID)
	if err != nil {
		return err
	}
	d.SetId(r.UID)
	return nil
}

func UpdateRole(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	if d.HasChange("version") || d.HasChange("name") || d.HasChange("description") || d.HasChange("permissions") || d.HasChange("org_id") {
		name := d.Get("name").(string)
		version, _ := d.Get("version").(int)
		description := d.Get("description").(string)
		orgId, _ := d.Get("org_id").(int)
		uid := d.Id()
		perms := permissions(d)
		policy := gapi.Role{
			OrgID:       int64(orgId),
			UID:         uid,
			Name:        name,
			Description: description,
			Version:     int64(version),
			Permissions: perms,
		}
		err := client.UpdateRole(policy)
		if err != nil {
			return err
		}
	}

	return nil
}

func DeleteRole(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)
	uid := d.Id()
	err := client.DeleteRole(uid)
	return err
}

func ExistsRole(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*gapi.Client)
	uid := d.Id()
	_, err := client.GetRole(uid)

	if err != nil && strings.HasPrefix(err.Error(), "status: 404") {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, err
}

func ImportRole(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	exists, err := ExistsRole(d, meta)
	if err != nil || !exists {
		return nil, errors.New(fmt.Sprintf("Error: Unable to import Grafana Role: %s.", err))
	}
	err = ReadRole(d, meta)
	if err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}
