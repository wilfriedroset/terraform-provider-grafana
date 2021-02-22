package grafana

import (
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
		Schema: map[string]*schema.Schema{
			"org_id": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
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
	rp := d.Get("permissions").([]interface{})

	perms := make([]gapi.Permission, 0)
	for _, p := range rp {
		ps := p.(map[string]interface{})
		perms = append(perms, gapi.Permission{
			Permission: ps["Permission"].(string),
			Scope:      ps["Scope"].(string),
		})
	}

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

func ReadPolicy(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func UpdatePolicy(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func DeletePolicy(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func ExistsPolicy(d *schema.ResourceData, meta interface{}) (bool, error) {
	return false, nil
}
