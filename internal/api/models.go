package jetbrains_space_api_client_go

import "net/http"

type Client struct {
	HostURL    string
	HTTPClient *http.Client
	Token      string
}

type Project struct {
	ID  string `json:"id"`
	Key struct {
		Key string `json:"key"`
	} `json:"key"`
	Name                     string      `json:"name"`
	Private                  bool        `json:"private"`
	Description              string      `json:"description"`
	Icon                     interface{} `json:"icon"`
	LatestRepositoryActivity interface{} `json:"latestRepositoryActivity"`
	CreatedAt                struct {
		Iso       string `json:"iso"`
		Timestamp int64  `json:"timestamp"`
	} `json:"createdAt"`
	Archived    bool           `json:"archived"`
	MemberTeams []ProjectTeams `json:"memberTeams"`
	Members     []struct {
		Profile struct {
			Username string `json:"username"`
		} `json:"profile"`
	} `json:"members"`
	AdminTeams []ProjectTeams `json:"adminTeams"`
	Admins     []struct {
		Username string `json:"username"`
	} `json:"adminProfiles"`
}

type ProjectTeams struct {
	Name string `json:"name"`
}

type ProjectRepos struct {
	Repos []struct {
		ID                        string      `json:"id"`
		Name                      string      `json:"name"`
		Description               string      `json:"description"`
		LatestActivity            interface{} `json:"latestActivity"`
		ProxyPushNotification     interface{} `json:"proxyPushNotification"`
		ProxyPushNotificationBody interface{} `json:"proxyPushNotificationBody"`
		State                     string      `json:"state"`
		InitProgress              interface{} `json:"initProgress"`
		ReadmeName                interface{} `json:"readmeName"`
		MonthlyActivity           interface{} `json:"monthlyActivity"`
		DefaultBranch             struct {
			Head string `json:"head"`
			Ref  string `json:"ref"`
		} `json:"defaultBranch"`
	} `json:"repos"`
}

type ProjectRoles struct {
	Team        string        `json:"team"`
	AddRoles    []interface{} `json:"addRoles"`
	RemoveRoles []interface{} `json:"removeRoles"`
}

type ProjectMembers struct {
	Profile     string        `json:"profile"`
	AddRoles    []interface{} `json:"addRoles"`
	RemoveRoles []interface{} `json:"removeRoles"`
}

type Repository struct {
	ID                        string      `json:"id"`
	Name                      string      `json:"name"`
	Description               string      `json:"description"`
	LatestActivity            interface{} `json:"latestActivity"`
	ProxyPushNotification     interface{} `json:"proxyPushNotification"`
	ProxyPushNotificationBody interface{} `json:"proxyPushNotificationBody"`
	State                     string      `json:"state"`
	InitProgress              interface{} `json:"initProgress"`
	ReadmeName                interface{} `json:"readmeName"`
	MonthlyActivity           interface{} `json:"monthlyActivity"`
	DefaultBranch             struct {
		Head string `json:"head"`
		Ref  string `json:"ref"`
	} `json:"defaultBranch"`
}

type CreateRepositoryData struct {
	Description   string `json:"description"`
	DefaultBranch string `json:"defaultBranch"`
	Initialize    bool   `json:"initialize"`
	DefaultSetup  bool   `json:"defaultSetup"`
}

type Projects struct {
	AllProjects []Project `json:"data"`
}

type AutomationJobs struct {
	Id         string `json:"id"`
	Name       string `json:"name"`
	Repository string `json:"repoName"`
}

type AllAutomationJobs struct {
	Data []struct {
		Id         string `json:"id"`
		Name       string `json:"name"`
		Repository string `json:"repoName"`
	} `json:"data"`
}
