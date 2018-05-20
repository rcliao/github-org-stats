package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

var (
	orgName      = flag.String("orgName", "", "Github organization name")
	repoPrefix   = flag.String("repoPrefix", "", "Repository prefix to look for")
	sinceTimeStr = flag.String("sinceTime", "", "Looking for commigs after sinceTime")
)

func main() {
	flag.Parse()
	layout := "2006-01-02T15:04 MST"
	sinceTime, err := time.Parse(layout, *sinceTimeStr)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("ACCESS_TOKEN")},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	organizationRepositories := getOrganizationRepositories(ctx, client, *orgName, *repoPrefix)

	var commits []github.RepositoryCommit
	for _, r := range organizationRepositories {
		commits = append(commits, getAllCommits(ctx, client, r, sinceTime)...)
	}
	json, _ := json.Marshal(commits)
	ioutil.WriteFile("commits.json", json, 0644)
	log.Println("Done writing commits to JSON")

	histogram := make(map[string]int)
	authorHistogram := make(map[string]int)
	location, err := time.LoadLocation("US/Pacific")
	if err != nil {
		log.Fatal(err)
	}
	for _, c := range commits {
		authorName := *c.Commit.Author.Name
		t := c.Commit.Author.Date.In(location)
		minute := t.Minute()
		if minute < 15 {
			minute = 0
		} else if minute >= 15 && minute < 30 {
			minute = 15
		} else if minute >= 30 && minute < 45 {
			minute = 30
		} else if minute >= 45 {
			minute = 45
		}
		bucket := fmt.Sprintf("%d:%d", t.Hour(), minute)
		histogram[bucket] = histogram[bucket] + 1
		authorHistogram[authorName] = authorHistogram[authorName] + 1
	}
	log.Println("date histogram:\n", histogram)
	plotHistogram(histogram)
}

func plotHistogram(rawValues map[string]int) {
	p, err := plot.New()
	if err != nil {
		panic(err)
	}
	p.Title.Text = "Histogram"
	p.Y.Label.Text = "# of commits"
	p.X.Label.Text = "Time range"

	keys := []string{}
	for k := range rawValues {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	values := make(plotter.Values, len(rawValues))
	labels := []string{}

	w := vg.Points(20)

	i := 0
	for _, k := range keys {
		values[i] = float64(rawValues[k])
		labels = append(labels, k)
		i++
	}

	barsA, err := plotter.NewBarChart(values, w)
	if err != nil {
		panic(err)
	}
	barsA.LineStyle.Width = vg.Length(0)
	barsA.Color = plotutil.Color(0)

	p.Add(barsA)
	p.NominalX(labels...)

	// Save the plot to a PNG file.
	if err := p.Save(4*vg.Inch, 4*vg.Inch, "bar.png"); err != nil {
		panic(err)
	}
}

func getOrganizationRepositories(ctx context.Context, client *github.Client, organizationName string, filterByName string) []github.Repository {
	opt := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 10},
	}
	var allRepos []*github.Repository
	for {
		repos, resp, err := client.Repositories.ListByOrg(ctx, organizationName, opt)
		if err != nil {
			log.Fatal("Has issue getting list of repos", err)
		}
		allRepos = append(allRepos, repos...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	var selectedRepos []github.Repository
	for _, r := range allRepos {
		if strings.Contains(*r.Name, filterByName) {
			selectedRepos = append(selectedRepos, *r)
		}
	}

	return selectedRepos
}

func getAllCommits(ctx context.Context, client *github.Client, r github.Repository, sinceTime time.Time) []github.RepositoryCommit {
	var allCommits []github.RepositoryCommit

	opt := &github.CommitsListOptions{
		Since:       sinceTime,
		ListOptions: github.ListOptions{PerPage: 100},
	}
	for {
		commits, resp, err := client.Repositories.ListCommits(ctx, *r.Owner.Login, *r.Name, opt)
		if err != nil {
			log.Fatal("Has issue getting list of commits", err)
		}
		for _, c := range commits {
			allCommits = append(allCommits, *c)
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allCommits
}
