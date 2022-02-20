package commercetools

import (
	"context"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/labd/commercetools-go-sdk/platform"
)

func resourceCustomerGroup() *schema.Resource {
	return &schema.Resource{
		Description: "A Customer can be a member of a customer group (for example reseller, gold member). " +
			"Special prices can be assigned to specific products based on a customer group.\n\n" +
			"See also the [Custome Group API Documentation](https://docs.commercetools.com/api/projects/customerGroups)",
		Create: resourceCustomerGroupCreate,
		Read:   resourceCustomerGroupRead,
		Update: resourceCustomerGroupUpdate,
		Delete: resourceCustomerGroupDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"key": {
				Description: "User-specific unique identifier for the customer group",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"version": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"name": {
				Description: "Unique within the project",
				Type:        schema.TypeString,
				Required:    true,
			},
		},
	}
}

func resourceCustomerGroupCreate(d *schema.ResourceData, m interface{}) error {
	client := getClient(m)
	var customerGroup *platform.CustomerGroup

	draft := platform.CustomerGroupDraft{
		GroupName: d.Get("name").(string),
		Key:       stringRef(d.Get("key")),
	}

	errorResponse := resource.Retry(1*time.Minute, func() *resource.RetryError {
		var err error

		customerGroup, err = client.CustomerGroups().Post(draft).Execute(context.Background())

		if err != nil {
			return handleCommercetoolsError(err)
		}
		return nil
	})

	if errorResponse != nil {
		return errorResponse
	}

	if customerGroup == nil {
		log.Fatal("No customer group")
	}

	d.SetId(customerGroup.ID)
	d.Set("version", customerGroup.Version)

	return resourceCustomerGroupRead(d, m)
}

func resourceCustomerGroupRead(d *schema.ResourceData, m interface{}) error {
	log.Printf("[DEBUG] Reading customer group from commercetools, with customer group id: %s", d.Id())

	client := getClient(m)

	customerGroup, err := client.CustomerGroups().WithId(d.Id()).Get().Execute(context.Background())

	if err != nil {
		if ctErr, ok := err.(platform.ErrorResponse); ok {
			if ctErr.StatusCode == 404 {
				d.SetId("")
				return nil
			}
		}
		return err
	}

	if customerGroup == nil {
		log.Print("[DEBUG] No customer group found")
		d.SetId("")
	} else {
		log.Print("[DEBUG] Found following customer group:")
		log.Print(stringFormatObject(customerGroup))

		d.Set("version", customerGroup.Version)
		d.Set("name", customerGroup.Name)
		d.Set("key", customerGroup.Key)
	}

	return nil
}

func resourceCustomerGroupUpdate(d *schema.ResourceData, m interface{}) error {
	client := getClient(m)
	customerGroup, err := client.CustomerGroups().WithId(d.Id()).Get().Execute(context.Background())
	if err != nil {
		return err
	}

	input := platform.CustomerGroupUpdate{
		Version: customerGroup.Version,
		Actions: []platform.CustomerGroupUpdateAction{},
	}

	if d.HasChange("name") {
		newName := d.Get("name").(string)
		input.Actions = append(
			input.Actions,
			&platform.CustomerGroupChangeNameAction{Name: newName})
	}

	if d.HasChange("key") {
		newKey := d.Get("key").(string)
		input.Actions = append(
			input.Actions,
			&platform.CustomerGroupSetKeyAction{Key: &newKey})
	}

	log.Printf(
		"[DEBUG] Will perform update operation with the following actions:\n%s",
		stringFormatActions(input.Actions))

	_, err = client.CustomerGroups().WithId(d.Id()).Post(input).Execute(context.Background())
	if err != nil {
		if ctErr, ok := err.(platform.ErrorResponse); ok {
			log.Printf("[DEBUG] %v: %v", ctErr, stringFormatErrorExtras(ctErr))
		}
		return err
	}

	return resourceCustomerGroupRead(d, m)
}

func resourceCustomerGroupDelete(d *schema.ResourceData, m interface{}) error {
	client := getClient(m)
	version := d.Get("version").(int)
	_, err := client.CustomerGroups().WithId(d.Id()).Delete().WithQueryParams(platform.ByProjectKeyCustomerGroupsByIDRequestMethodDeleteInput{
		Version: version,
	}).Execute(context.Background())
	if err != nil {
		log.Printf("[ERROR] Error during deleting customer group resource %s", err)
		return nil
	}
	return nil
}
