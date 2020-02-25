package config

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/google/go-github/v29/github"
	"github.com/hashicorp/go-getter"
	"github.com/rancher/channelserver/pkg/model"
	"github.com/rancher/wrangler/pkg/data/convert"
	"sigs.k8s.io/yaml"
)

func getURLs(ctx context.Context, urls ...string) ([]byte, error) {
	var (
		bytes []byte
		err   error
	)
	for _, url := range urls {
		bytes, err = get(ctx, url)
		if err == nil {
			break
		}
	}

	return bytes, err
}

func get(ctx context.Context, url string) ([]byte, error) {
	content, err := ioutil.ReadFile(url)
	if err == nil {
		return content, nil
	}

	tmp, err := ioutil.TempFile("", "channel-config*.yaml")
	if err != nil {
		return nil, err
	}
	defer func() {
		tmp.Close()
		os.Remove(tmp.Name())
	}()

	err = getter.GetFile(tmp.Name(), url, getter.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(tmp)
}

func GetConfig(ctx context.Context, subKey string, configURLs ...string) (*model.ChannelsConfig, error) {
	var (
		data   = map[string]interface{}{}
		config = &model.ChannelsConfig{}
	)

	content, err := getURLs(ctx, configURLs...)
	if err != nil {
		return nil, err
	}

	if subKey == "" {
		return config, yaml.Unmarshal(content, config)
	}

	if err := yaml.Unmarshal(content, &data); err != nil {
		return nil, err
	}
	data, _ = data[subKey].(map[string]interface{})
	if data == nil {
		return nil, fmt.Errorf("failed to find key %s in config", subKey)
	}
	return config, convert.ToObj(data, config)
}

func GetReleases(ctx context.Context, client *github.Client, owner, repo string) ([]string, error) {
	var (
		opt         = &github.ListOptions{}
		allReleases []string
	)

	for {
		releases, resp, err := client.Repositories.ListReleases(ctx, owner, repo, opt)
		if err != nil {
			return nil, err
		}
		for _, release := range releases {
			if release.TagName != nil {
				allReleases = append(allReleases, *release.TagName)
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allReleases, nil
}
