# Jira Automation Helper
Since Jira is unable to update fields in ADF format with Automations, this can be used as an alternative.  

### Attention
This is a quick'n'dirty solution and should not be run without securing your stuff.  

### Config File
```yml
APIKey: "your-secret-api-key"
```

Then call the service via POST at `/v1/issue/` with the following payload.

### Examples

Authorization header must match the value in your config file

```json
{
    "id": "JIRA-200",
    "type": "outward",
    "typevalue": "implements",
    "fields": ["description", "customfield_1", "customfield_2", "customfield_3", "customfield_4", "customfield_5", "customfield_6"],
    "api_key": "YOUR_JIRA_API_KEY",
    "email": "YOUR_JIRA_EMAIL",
    "base_url": "https://YOUR-COMPANY.atlassian.net"
}
```
