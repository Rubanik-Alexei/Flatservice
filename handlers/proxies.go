package handlers

import (
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/corpix/uarand"
)

func GetProxiesList() []string {
	url := string("https://proxylist.geonode.com/api/proxy-list?limit=200&page=1&sort_by=lastChecked&sort_type=desc")
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)

	req.Header.Set("User-Agent", uarand.GetRandom())
	resp, err := client.Do(req)
	//cont, err := soup.GetWithClient(url,client)
	if err != nil {
		return []string{}
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []string{}
	}
	proxies := []string{}
	res := string(body)
	reip := regexp.MustCompile(`ip":"\s*(.*?)\s*","anonymityLevel`)
	re_port := regexp.MustCompile(`port":"\s*(.*?)\s*","pr`)
	ips := reip.FindAllStringSubmatch(res, -1)
	ports := re_port.FindAllStringSubmatch(res, -1)
	for i := 0; i < len(ips); i++ {
		proxies = append(proxies, ips[i][1]+":"+ports[i][1])
	}
	return proxies
}
