package internal

import "net/url"

func SubURL(base *url.URL, name string) (*url.URL, error) {
	rel, err := url.Parse(name)
	if err != nil {
		return nil, err
	}

	u := base.ResolveReference(rel)

	// also merge query params
	if base.RawQuery != "" {
		bq := base.Query()
		rq := rel.Query()

		for k := range rq {
			bq.Set(k, rq.Get(k))
		}

		u.RawQuery = bq.Encode()
	}

	return u, nil
}
