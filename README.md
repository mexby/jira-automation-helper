# Jira Automation Helper
Since Jira is unable to update fields in ADF format with Automations, this can be used as an alternative.  

### Attention
This is a quick'n'dirty solution and should not be run without securing your stuff.  

### Config File
```yml
JiraAPIKey: "your-secret-api-key"
JiraEmail: "your-email@domain.com"
JiraBaseURL: "https://your-company.atlassian.net"
```

Then call the service via `/v1/issue/{id}/{type}/{typevalue}/{fields}`.

### Examples
> /v1/issue/JIRA-123/outward/implements/description,customfield_12345  

 This updates the fields Description and customfield_12345 of all linked issues of type "implements" of issue JIRA-123

 > /v1/issue/JIRA-123/inward/is implemented by/description,customfield_12345  

 This updates the fields Description and customfield_12345 of all linked issues of type "is implemented by" of issue JIRA-123