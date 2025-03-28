package models

type FilterData struct {
	Categories []FiltersCategory `json:"categories"`
}

type FiltersCategory struct {
	CategoryName string          `json:"categoryName"`
	CategoryID   string          `json:"categoryId"`
	CategoryData []CategoryGroup `json:"categoryData"`
}

type CategoryGroup struct {
	Group string       `json:"group,omitempty"`
	Data  []FilterItem `json:"data"`
}

type FilterItem struct {
	Id          string `json:"id"`
	FilterLabel string `json:"filterLabel"`
	CardLabel   string `json:"cardLabel,omitempty"`
	Color       string `json:"color,omitempty"`
	Icon        string `json:"icon,omitempty"`
}

var (
	AnsibleIcon   = "/apps/frontend-assets/console-landing/ansible.svg"
	OpenShiftIcon = "/apps/frontend-assets/console-landing/openshift.svg"
	RHELIcon      = "/apps/frontend-assets/learning-resources/RHEL-icon.svg"
	RedHatIcon    = "/apps/frontend-assets/learning-resources/RH-icon.svg"

	FrontendFilters FilterData = FilterData{
		Categories: []FiltersCategory{
			{
				CategoryName: "Product families",
				CategoryID:   "product-families",
				CategoryData: []CategoryGroup{{
					Group: "Platforms",
					Data: []FilterItem{
						{Id: "ansible", CardLabel: "Ansible", FilterLabel: "Ansible", Icon: AnsibleIcon},
						{Id: "openshift", CardLabel: "OpenShift", FilterLabel: "OpenShift", Icon: OpenShiftIcon},
						{Id: "rhel", CardLabel: "RHEL", FilterLabel: "RHEL (Red Hat Enterprise Linux)", Icon: RHELIcon},
					},
				},
					{
						Group: "Console-wide services",
						Data: []FilterItem{
							{Id: "iam", CardLabel: "IAM", FilterLabel: "IAM (Identity & Access Management)", Icon: RedHatIcon},
							{Id: "settings", CardLabel: "Settings", FilterLabel: "Settings", Icon: RedHatIcon},
							{Id: "subscriptions-services", CardLabel: "Subscriptions Services", FilterLabel: "Subscriptions Services", Icon: RedHatIcon},
						},
					},
				},
			},
			{
				CategoryName: "Content type",
				CategoryID:   "content",
				CategoryData: []CategoryGroup{{
					Data: []FilterItem{
						{Id: "documentation", CardLabel: "Documentation", FilterLabel: "Documentation", Color: "orange"},
						{Id: "learningPath", CardLabel: "Learning path", FilterLabel: "Learning path", Color: "cyan"},
						{Id: "quickstart", CardLabel: "Quick start", FilterLabel: "Quick start", Color: "green"},
						{Id: "otherResource", CardLabel: "Other", FilterLabel: "Other", Color: "purple"},
					},
				}},
			},
			{
				CategoryName: "Use case",
				CategoryID:   "use-case",
				CategoryData: []CategoryGroup{{
					Data: []FilterItem{
						{Id: "automation", CardLabel: "Automation", FilterLabel: "Automation"},
						{Id: "clusters", CardLabel: "Clusters", FilterLabel: "Clusters"},
						{Id: "containers", CardLabel: "Containers", FilterLabel: "Containers"},
						{Id: "data-services", CardLabel: "Data services", FilterLabel: "Data services"},
						{Id: "deploy", CardLabel: "Deploy", FilterLabel: "Deploy"},
						{Id: "identity-and-access", CardLabel: "Identity and access", FilterLabel: "Identity and access"},
						{Id: "images", CardLabel: "Images", FilterLabel: "Images"},
						{Id: "infrastructure", CardLabel: "Infrastructure", FilterLabel: "Infrastructure"},
						{Id: "observability", CardLabel: "Observability", FilterLabel: "Observability"},
						{Id: "security", CardLabel: "Security", FilterLabel: "Security"},
						{Id: "spend-management", CardLabel: "Spend management", FilterLabel: "Spend management"},
						{Id: "system-configuration", CardLabel: "System configuration", FilterLabel: "System configuration"},
					},
				}},
			},
		},
	}
)
