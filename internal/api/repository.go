package jetbrains_space_api_client_go

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type ProtectedBranches struct {
	ProtectedBranches []ProtectedBranchesReq `json:"protectedBranches"`
}

type ProtectedBranchesSettings struct {
	Version           string                 `json:"version"`
	ProtectedBranches []ProtectedBranchesReq `json:"protectedBranches"`
}
type ProtectedBranchesReq struct {
	Pattern        []string                     `json:"pattern"`
	AllowCreate    []string                     `json:"allowCreate"`
	AllowPush      []string                     `json:"allowPush"`
	AllowDelete    []string                     `json:"allowDelete"`
	AllowForcePush []string                     `json:"allowForcePush"`
	QualityGate    ProtectedBranchesQualityGate `json:"qualityGate"`
}

type ProtectedBranchesPost struct {
	Settings ProtectedBranchesSettings `json:"settings"`
}

type ProtectedBranchesPostSettings struct {
	Version           string                 `json:"version"`
	ProtectedBranches []ProtectedBranchesReq `json:"protectedBranches"`
}

type ProtectedBranchesQualityGate struct {
	Approvals []ProtectedBranchesResultApprovals `json:"approvals"`
}

type ProtectedBranchesResultApprovals struct {
	ApprovedBy   []string `json:"approvedBy"`
	MinApprovals int      `json:"minApprovals"`
}

func (c *Client) GetRepository(repositoryName, projectId string) (Repository, error) {
	projectRepos, err := c.getProjectRepos(projectId)
	if err != nil {
		return Repository{}, err
	}
	for _, repo := range projectRepos.Repos {
		if repo.Name == repositoryName {
			return repo, nil
		}
	}
	return Repository{}, fmt.Errorf("repository %s not found", repositoryName)
}

func (c *Client) CreateRepository(repositoryName string, projectId string, data CreateRepositoryData) (Repository, error) {
	bytesData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", fmt.Sprintf("%s%s/id:%s/repositories/%s", c.HostURL, baseAPIEndpoint, projectId, repositoryName), bytes.NewBuffer(bytesData))
	if err != nil {
		return Repository{}, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return Repository{}, err
	}

	repository := Repository{}
	err = json.Unmarshal(body, &repository)
	if err != nil {
		return Repository{}, err
	}

	return repository, nil
}

func (c *Client) UpdateRepository(projectId, name string, data CreateRepositoryData) (Repository, error) {
	bytesData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", fmt.Sprintf("%s%s/id:%s/repositories/%s/settings", c.HostURL, baseAPIEndpoint, projectId, name), bytes.NewBuffer(bytesData))
	if err != nil {
		return Repository{}, err
	}

	body, err := c.doRequest(req)
	if err != nil {
		return Repository{}, err
	}

	Repository := Repository{}
	err = json.Unmarshal(body, &Repository)
	if err != nil {
		return Repository, err
	}

	return Repository, nil
}

func (c *Client) UpdateRepositoryDescription(projectId, name string, description string) (string, error) {
	desc := map[string]string{
		"description": description,
	}
	bytesData, _ := json.Marshal(desc)
	req, err := http.NewRequest("POST", fmt.Sprintf("%s%s/id:%s/repositories/%s/description", c.HostURL, baseAPIEndpoint, projectId, name), bytes.NewBuffer(bytesData))
	if err != nil {
		return "", fmt.Errorf("Problem!")
	}

	_, err = c.doRequest(req)
	if err != nil {
		return "", fmt.Errorf("Problem!2")
	}

	return description, nil
}

func (c *Client) UpdateRepositoryDefaultBranch(projectId, name string, branch string) (string, error) {
	desc := map[string]string{
		"branch": branch,
	}
	bytesData, _ := json.Marshal(desc)
	req, err := http.NewRequest("POST", fmt.Sprintf("%s%s/id:%s/repositories/%s/default-branch", c.HostURL, baseAPIEndpoint, projectId, name), bytes.NewBuffer(bytesData))
	if err != nil {
		return "", fmt.Errorf("Problem initiating request to update repository branch via API! " + err.Error())
	}

	_, err = c.doRequest(req)
	if err != nil {
		return "", fmt.Errorf("Problem updating repository branch via API " + err.Error())
	}

	return branch, nil
}

func (c *Client) DeleteRepository(projectId, repositoryName string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s%s/id:%s/repositories/%s", c.HostURL, baseAPIEndpoint, projectId, repositoryName), nil)
	if err != nil {
		return err
	}

	_, err = c.doRequest(req)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) DeleteRepositoryProtectedBranches(projectId, name string) error {

	settings := map[string]interface{}{
		"settings": map[string]interface{}{
			"protectedBranches": nil,
		},
	}

	bytesData, _ := json.Marshal(settings)
	req, err := http.NewRequest("POST", fmt.Sprintf("%s%s/id:%s/repositories/%s/settings", c.HostURL, baseAPIEndpoint, projectId, name), bytes.NewBuffer(bytesData))
	if err != nil {
		return fmt.Errorf("Problem initiating request to deleted repository branch via API! " + err.Error())
	}

	_, err = c.doRequest(req)
	if err != nil {
		return fmt.Errorf("Problem getting response from protected branches " + err.Error())
	}

	return nil

}

func (c *Client) UpdateRepoProtectedBranches(data ProtectedBranchesPost, ProjectID string, Repository string) (ProtectedBranches, error) {

	jsonData, err := json.Marshal(data)
	if err != nil {
		return ProtectedBranches{}, fmt.Errorf("Problem converting request data to valid json")
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s%s/id:%s/repositories/%s/settings", c.HostURL, baseAPIEndpoint, ProjectID, Repository), bytes.NewBuffer(jsonData))
	if err != nil {
		return ProtectedBranches{}, fmt.Errorf("Problem initiating request to update repository branch via API! " + err.Error())
	}
	_, err = c.doRequest(req)
	if err != nil {
		return ProtectedBranches{}, fmt.Errorf("Problem getting response from protected branches (Updating) " + err.Error())

	}
	// API doesnt return validation. Run a get.

	protected, err := c.GetRepoProtectedBranches(ProjectID, Repository)
	if err != nil {
		return ProtectedBranches{}, err
	}

	return protected, nil

}

func (c *Client) GetRepoProtectedBranches(ProjectID string, Repository string) (ProtectedBranches, error) {

	req, err := http.NewRequest("GET", fmt.Sprintf("%s%s/id:%s/repositories/%s/settings?$fields=protectedBranches(allowCreate,allowDelete,allowForcePush,allowPush,pattern,qualityGate(approvals(approvedBy,minApprovals)))", c.HostURL, baseAPIEndpoint, ProjectID, Repository), nil)
	if err != nil {
		return ProtectedBranches{}, fmt.Errorf("Problem setting up new http request; " + err.Error())
	}
	body, err := c.doRequest(req)
	if err != nil {
		return ProtectedBranches{}, fmt.Errorf("Problem getting repository branch settings via API! " + err.Error())
	}

	var protected ProtectedBranches
	err = json.Unmarshal(body, &protected)
	if err != nil {
		return ProtectedBranches{}, err
	}

	return protected, nil

}
