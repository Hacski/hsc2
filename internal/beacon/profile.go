package beacon

import (
	"fmt"
	"math/rand"
	"time"
)

type Profile struct {
	Name      string        `json:"name"`
	Interval  time.Duration `json:"interval"`
	Jitter    float64       `json:"jitter"`
	Transport string        `json:"transport"`
	URI       []string      `json:"uri"`
	UserAgent string        `json:"user_agent"`
	Headers   map[string]string `json:"headers"`
	SleepCaps map[string]time.Duration `json:"sleep_caps"`
}

func Default() Profile {
	return Profile{
		Name:      "default",
		Interval:  60 * time.Second,
		Jitter:    0.05,
		Transport: "https",
		URI:       []string{"/index.html", "/profile.php", "/wp-admin"},
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
		Headers:   map[string]string{"Accept-Language": "en-US,en;q=0.9"},
	}
}

func Webserver() Profile {
	return Profile{
		Name:      "webserver",
		Interval:  30 * time.Second,
		Jitter:    0.2,
		Transport: "http",
		URI:       []string{"/static/js/main.js", "/assets/app.js", "/js/jquery.min.js"},
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		Headers:   map[string]string{"Accept": "application/javascript", "Referer": "/"},
	}
}

func DnsTunneling() Profile {
	return Profile{
		Name:      "dnstunnel",
		Interval:  5 * time.Second,
		Jitter:    0.4,
		Transport: "dns",
		URI:       []string{""},
		UserAgent: "",
	}
}

func (p Profile) NextSleep() time.Duration {
	if p.Jitter <= 0 {
		return p.Interval
	}
	delta := time.Duration(float64(p.Interval) * p.Jitter * rand.Float64())
	if rand.Intn(2) == 0 {
		return p.Interval - delta
	}
	return p.Interval + delta
}

func (p Profile) Validate() error {
	if p.Interval <= 0 {
		return fmt.Errorf("interval must be positive")
	}
	if p.Jitter < 0 || p.Jitter > 1 {
		return fmt.Errorf("jitter must be in [0,1]")
	}
	if len(p.URI) == 0 {
		return fmt.Errorf("profile must define at least one uri")
	}
	return nil
}

func (p Profile) PickURI() string {
	return p.URI[rand.Intn(len(p.URI))]
}
