package jetbrains_space_api_client_go

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func (c *Client) GetProjects() (Projects, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/http/projects", c.HostURL), nil)
	if err != nil {
		return Projects{}, err
	}
	body, err := c.doRequest(req)
	if err != nil {
		return Projects{}, err
	}
	projects := Projects{}
	err = json.Unmarshal(body, &projects)
	if err != nil {
		return Projects{}, err
	}

	return projects, nil

}

func (c *Client) GetProject(id string) (Project, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/http/projects/id:%s?$fields=id,archived,createdAt,description,icon,key,latestRepositoryActivity,name,private,memberTeams(name)", c.HostURL, id), nil)
	if err != nil {
		return Project{}, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return Project{}, err
	}

	project := Project{}
	err = json.Unmarshal(body, &project)
	if err != nil {
		return Project{}, err
	}

	return project, nil
}

func (c *Client) getProjectRepos(projectId string) (ProjectRepos, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s%s/id:%s?$fields=repos", c.HostURL, baseApiEndpoint, projectId), nil)
	if err != nil {
		return ProjectRepos{}, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return ProjectRepos{}, err
	}

	project := ProjectRepos{}
	err = json.Unmarshal(body, &project)
	if err != nil {
		return ProjectRepos{}, err
	}

	return project, nil
}

func (c *Client) CreateProject(name string) (Project, error) {
	data := new(struct {
		Key struct {
			Key string `json:"key"`
		} `json:"key"`
		Name string `json:"name"`
	})
	data.Name = name
	data.Key.Key = strings.ToUpper(strings.ReplaceAll(name, " ", "-"))
	bytesData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", fmt.Sprintf("%s%s", c.HostURL, baseApiEndpoint), bytes.NewBuffer(bytesData))
	if err != nil {
		return Project{}, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return Project{}, err
	}

	project := Project{}
	err = json.Unmarshal(body, &project)
	if err != nil {
		return Project{}, err
	}

	return project, nil
}

func (c *Client) UpdateProject(id string, project Project) (Project, error) {
	data := new(struct {
		Name string `json:"name"`
	})
	data.Name = project.Name
	bytesData, _ := json.Marshal(data)
	req, err := http.NewRequest("PATCH", fmt.Sprintf("%s%s/id:%s", c.HostURL, baseApiEndpoint, id), bytes.NewBuffer(bytesData))
	if err != nil {
		return Project{}, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return Project{}, err
	}

	updatedProject := Project{}
	err = json.Unmarshal(body, &updatedProject)
	if err != nil {
		return Project{}, err
	}

	return updatedProject, nil
}

func (c *Client) DeleteProject(id string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s%s/id:%s", c.HostURL, baseApiEndpoint, id), nil)
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) MapTeamToProjectRole(data ProjectRoles, projectID string) error {

	jsonData, err := json.Marshal(data)
	jsonData = bytes.Replace(jsonData, []byte("\"\""), []byte(""), 1)
	if err != nil {
		return fmt.Errorf("Problem converting request data to valid json")
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s%s/id:%s/people/teams/update", c.HostURL, baseApiEndpoint, projectID), bytes.NewBuffer(jsonData))
	req.Header.Add("Accept", "Application/json")
	req.Header.Add("Content-Type", "Application/Json")
	if err != nil {
		return fmt.Errorf("Problem initiating request to update project roles via API! " + err.Error())
	}
	_, err = c.doRequest(req)
	if err != nil {
		return fmt.Errorf("Problem getting response from project roles (Updating) " + string(jsonData) + err.Error())

	}

	return nil

}

func (c *Client) GetTeamToProjectRole(projectID string) ([]ProjectTeams, error) {
	projectSettings, err := c.GetProject(projectID)
	if err != nil {
		return []ProjectTeams{}, fmt.Errorf("Problem getting project settings" + projectID + " " + err.Error())
	}

	return projectSettings.MemberTeams, nil

}
