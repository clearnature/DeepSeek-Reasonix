package responses

import "testing"

func TestDomainOf(t *testing.T) {
	cases := map[string]string{
		"https://reuters.com/world/xyz":    "reuters.com",
		"http://www.bbc.co.uk/news":        "bbc.co.uk",
		"https://sub.domain.gov.cn/page":   "sub.domain.gov.cn",
		"https://zh.wikipedia.org/wiki/AI": "zh.wikipedia.org",
		"not a url":                        "",
		"":                                 "",
	}
	for in, want := range cases {
		if got := domainOf(in); got != want {
			t.Errorf("domainOf(%q)=%q want %q", in, got, want)
		}
	}
}

func TestIsAuthorityDomain(t *testing.T) {
	if !isAuthorityDomain("nasa.gov") || !isAuthorityDomain("university.edu.cn") || !isAuthorityDomain("army.mil") {
		t.Fatal("authority domains must be recognized")
	}
	if isAuthorityDomain("example.com") || isAuthorityDomain("junk.net") {
		t.Fatal("non-authority domains must not be recognized")
	}
}

func TestScoreAndFilterSources(t *testing.T) {
	e := &KnowledgeEntry{Sources: []Source{
		{URL: "https://reuters.com/world/a"},                      // whitelist 0.4 + cross 0.3 = 0.7
		{URL: "https://www.gov.cn/policy/b"},                      // authority 0.2 + cross 0.3 = 0.5
		{URL: "https://junk-blog.example.com/post?utm_source=ad"}, // spam hint -0.1 + cross 0.3 = 0.2
	}}
	ScoreAndTagSources(e)

	if e.Sources[0].Credibility < 0.6 || e.Sources[0].Domain != "reuters.com" {
		t.Fatalf("reuters should score >=0.6, got %.2f (%s)", e.Sources[0].Credibility, e.Sources[0].Domain)
	}
	if e.Sources[1].Credibility < 0.45 {
		t.Fatalf("gov.cn should score >=0.45, got %.2f", e.Sources[1].Credibility)
	}
	if e.Sources[2].Credibility > 0.35 {
		t.Fatalf("spam source should score <=0.35, got %.2f", e.Sources[2].Credibility)
	}

	kept := FilterSources(e.Sources, 0.5)
	if len(kept) != 2 {
		t.Fatalf("want 2 kept, got %d: %#v", len(kept), kept)
	}
	if kept[0].URL == e.Sources[2].URL {
		t.Fatal("spam source must be filtered out")
	}
}

func TestSpamDomainPenalized(t *testing.T) {
	e := &KnowledgeEntry{Sources: []Source{
		{URL: "https://medium.com/clickbait"},
		{URL: "https://reuters.com/real"},
	}}
	ScoreAndTagSources(e)
	if e.Sources[0].Credibility >= e.Sources[1].Credibility {
		t.Fatalf("spam domain must score below whitelist: %.2f vs %.2f", e.Sources[0].Credibility, e.Sources[1].Credibility)
	}
}
