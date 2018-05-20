# Github Organization Commit Stats

This CLI app retrieves all commits from Github organization with a repository prefix search and draw out a bar chart based on the time.

## Usage

Set up a environment variable containing the token:

```
export ACCESS_TOKEN={your_github_access_token}
```

```
go run main.go --orgName csula-students --repoPrefix cs-3220-spring-2018-final --sinceTime "2018-05-19T16:00 PST" && open bar.png
```
