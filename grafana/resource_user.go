package grafana

import (
	"errors"
	"fmt"
	"strconv"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform/helper/schema"
)

func ResourceUser() *schema.Resource {
	return &schema.Resource{
		Create: CreateUser,
		Read:   ReadUser,
		Update: UpdateUser,
		Delete: DeleteUser,
		Exists: ExistsUser,
		Importer: &schema.ResourceImporter{
			State: ImportUser,
		},
		Schema: map[string]*schema.Schema{
			"email": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"login": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"password": {
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
			},
			"is_admin": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"roles": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func CreateUser(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)
	user := gapi.User{
		Email:    d.Get("email").(string),
		Name:     d.Get("name").(string),
		Login:    d.Get("login").(string),
		Password: d.Get("password").(string),
	}
	id, err := client.CreateUser(user)
	if err != nil {
		return err
	}
	if d.HasChange("is_admin") {
		err = client.UpdateUserPermissions(id, d.Get("is_admin").(bool))
		if err != nil {
			return err
		}
	}
	err = UpdateUserRoles(d, meta)
	if err != nil {
		return err
	}
	d.SetId(strconv.FormatInt(id, 10))
	return ReadUser(d, meta)
}

func ReadUser(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)
	id, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return err
	}
	user, err := client.User(id)
	if err != nil {
		return err
	}
	d.Set("email", user.Email)
	d.Set("name", user.Name)
	d.Set("login", user.Login)
	d.Set("is_admin", user.IsAdmin)
	if err := ReadUserRoles(d, meta); err != nil {
		return err
	}
	return nil
}

func ReadUserRoles(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)
	userID, _ := strconv.ParseInt(d.Id(), 10, 64)
	userRoles, err := client.GetUserRoles(userID)
	if err != nil {
		return err
	}
	var roles []string
	for _, ur := range userRoles {
		roles = append(roles, ur.UID)
	}

	return d.Set("roles", roles)
}

func UpdateUser(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)
	id, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return err
	}
	u := gapi.User{
		ID:    id,
		Email: d.Get("email").(string),
		Name:  d.Get("name").(string),
		Login: d.Get("login").(string),
	}
	err = client.UserUpdate(u)
	if err != nil {
		return err
	}
	if d.HasChange("password") {
		err = client.UpdateUserPassword(id, d.Get("password").(string))
		if err != nil {
			return err
		}
	}
	if d.HasChange("is_admin") {
		err = client.UpdateUserPermissions(id, d.Get("is_admin").(bool))
		if err != nil {
			return err
		}
	}
	err = UpdateUserRoles(d, meta)
	if err != nil {
		return err
	}
	return ReadUser(d, meta)
}

func UpdateUserRoles(d *schema.ResourceData, meta interface{}) error {
	stateRoles, configRoles, err := collectRoles(d)
	if err != nil {
		return err
	}
	changes := roleChanges(stateRoles, configRoles)
	userID, _ := strconv.ParseInt(d.Id(), 10, 64)
	return applyRoleChangesToUser(meta, userID, changes)
}

func DeleteUser(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)
	id, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return err
	}
	return client.DeleteUser(id)
}

func ExistsUser(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*gapi.Client)
	userId, _ := strconv.ParseInt(d.Id(), 10, 64)
	_, err := client.User(userId)
	if err != nil {
		return false, err
	}
	return true, nil
}

func ImportUser(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	exists, err := ExistsUser(d, meta)
	if err != nil || !exists {
		return nil, errors.New(fmt.Sprintf("Error: Unable to import Grafana User: %s.", err))
	}
	err = ReadUser(d, meta)
	if err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}

func applyRoleChangesToUser(meta interface{}, userId int64, changes []RoleChange) error {
	var err error
	client := meta.(*gapi.Client)
	for _, change := range changes {
		r := change.UID
		switch change.Type {
		case AddRole:
			err = client.NewUserRole(userId, r)
		case RemoveRole:
			err = client.DeleteUserRole(userId, r)
		}
		if err != nil {
			return errors.New(fmt.Sprintf("Error with %s %v", r, err))
		}
	}
	return nil
}
