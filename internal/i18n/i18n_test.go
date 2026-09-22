package i18n

import "testing"

func TestSameKeys(t *testing.T) {
	for k := range es {
		if _, ok := en[k]; !ok {
			t.Errorf("falta %q en en", k)
		}
	}
	for k := range en {
		if _, ok := es[k]; !ok {
			t.Errorf("falta %q en es", k)
		}
	}
}

func TestPick(t *testing.T) {
	cases := map[string]Lang{
		"":                        ES,
		"es-AR,es;q=0.9":          ES,
		"en-US,en;q=0.9":          EN,
		"en-GB":                   EN,
		"pt-BR,pt;q=0.9":          ES,
		"fr-FR,en;q=0.8,es;q=0.9": ES,
		"fr-FR,es;q=0.5,en;q=0.8": EN,
		"*":                       ES,
	}
	for in, want := range cases {
		if got := Pick(in); got != want {
			t.Errorf("Pick(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestT(t *testing.T) {
	if got := T(ES, "sharing_n", "Martín", 3); got != "Martín te comparte 3 archivos" {
		t.Fatalf("got %q", got)
	}
	if got := T(EN, "sharing_n", "Martín", 3); got != "Martín is sharing 3 files with you" {
		t.Fatalf("got %q", got)
	}
}

func TestSize(t *testing.T) {
	cases := []struct {
		l    Lang
		n    int64
		want string
	}{
		{ES, 0, "0 B"}, {ES, 999, "999 B"}, {ES, 1300, "1,3 KB"},
		{ES, 4_200_000, "4,2 MB"}, {ES, 980_000_000, "980 MB"},
		{ES, 1_200_000_000, "1,2 GB"}, {EN, 1_200_000_000, "1.2 GB"},
	}
	for _, c := range cases {
		if got := Size(c.l, c.n); got != c.want {
			t.Errorf("Size(%s,%d) = %q, want %q", c.l, c.n, got, c.want)
		}
	}
}
