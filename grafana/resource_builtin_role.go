package grafana

import (
	"errors"
	"fmt"
	"log"
	
	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func ResourceBuiltInRole() *schema.Resource {
	return &schema.Resource{
		Create: CreateBuiltInRole,
		Update: UpdateBuiltInRole,
		Read:   ReadBuiltInRole,
		Delete: DeleteBuiltInRole,
		Exists: ExistsBuiltInRole,
		Importer: &schema.ResourceImporter{
			State: ImportBuiltInRole,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"Grafana Admin", "Admin", "Editor", "Viewer"}, false),
			},
			"roles": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func CreateBuiltInRole(d *schema.ResourceData, meta interface{}) error {
	name := d.Get("name").(string)
	err := UpdateBuiltInRoles(d, meta)
	if err != nil {
		return err
	}
	d.SetId(name)
	return nil
}

func UpdateBuiltInRoles(d *schema.ResourceData, meta interface{}) error {
	stateRoles, configRoles, err := collectRoles(d)
	if err != nil {
		return err
	}
	//compile the list of differences between current state and config
	changes := roleChanges(stateRoles, configRoles)
	brName := d.Get("name").(string)
	//now we can make the corresponding updates so current state matches config
	return applyRoleChangesToBuiltInRole(meta, brName, changes)
}

func ReadBuiltInRole(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)
	brName := d.Id()
	builtInRoles, err := client.GetBuiltInRoles()

	if err != nil {
		return err
	}

	brRole := builtInRoles[brName]
	if builtInRoles[brName] == nil {
		log.Printf("[WARN] removing built-in role %s from state because it no longer exists in grafana", d.Id())
		d.SetId("")
		return nil
	}

	var roles []string
	for _, br := range brRole {
		roles = append(roles, br.UID)
	}

	err = d.Set("roles", roles)
	if err != nil {
		return err
	}

	err = d.Set("name", brName)
	if err != nil {
		return err
	}
	d.SetId(brName)
	return nil
}

func UpdateBuiltInRole(d *schema.ResourceData, meta interface{}) error {
	if d.HasChange("roles") {
		if err := UpdateBuiltInRoles(d, meta); err != nil {
			return err
		}
	}

	return nil
}

func DeleteBuiltInRole(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)
	brName := d.Id()

	for _, r := range d.Get("roles").([]interface{}) {
		uid := r.(string)
		err := client.DeleteBuiltInRole(gapi.BuiltRole{RoleUID: uid, BuiltinRole: brName})
		if err != nil {
			return err
		}
	}
	d.SetId("")
	return nil
}

func ExistsBuiltInRole(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*gapi.Client)
	brName := d.Id()
	brRoles, err := client.GetBuiltInRoles()

	if err != nil {
		return false, err
	}
	if brRoles[brName] == nil {
		return false, nil
	}

	return true, err
}

func ImportBuiltInRole(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	exists, err := ExistsBuiltInRole(d, meta)
	if err != nil || !exists {
		return nil, errors.New(fmt.Sprintf("Error: Unable to import Grafana Built-In Role: %s.", err))
	}
	err = ReadBuiltInRole(d, meta)
	if err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}

func applyRoleChangesToBuiltInRole(meta interface{}, name string, changes []RoleChange) error {
	var err error
	client := meta.(*gapi.Client)
	for _, change := range changes {
		br := gapi.BuiltRole{BuiltinRole: name, RoleUID: change.UID}
		switch change.Type {
		case AddRole:
			_, err = client.NewBuiltInRole(br)
		case RemoveRole:
			err = client.DeleteBuiltInRole(br)
		}
		if err != nil {
			return errors.New(fmt.Sprintf("Error with %s %v", name, err))
		}
	}
	return nil
}
