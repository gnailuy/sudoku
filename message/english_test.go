package message

import "testing"

func TestEnglishCatalogRendersCanonicalNames(t *testing.T) {
	catalog, err := NewEnglishCatalog()
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"difficulty.easy.name":             "Easy",
		"difficulty.evil.name":             "Evil",
		"strategy.naked-single.name":       "Naked Single",
		"strategy.pointing-pair.name":      "Pointing Pairs / Box-Line Reduction",
		"strategy.unique-rectangle-4.name": "Unique Rectangle Type 4",
		"hint.value-correction.title":      "Fix this value",
		"hint.candidate-correction.title":  "Fix this candidate cell",
	}
	for id, want := range cases {
		descriptor, err := catalog.Render(id)
		if err != nil {
			t.Fatalf("Render(%q): %v", id, err)
		}
		if descriptor.ID != id || descriptor.Text != want || len(descriptor.Arguments) != 0 {
			t.Fatalf("Render(%q) = %#v, want text %q and no arguments", id, descriptor, want)
		}
	}
}

func TestEnglishCatalogCoversStableNameIDs(t *testing.T) {
	catalog, err := NewEnglishCatalog()
	if err != nil {
		t.Fatal(err)
	}
	for _, domainIDs := range []struct {
		domain string
		ids    []string
	}{
		{"difficulty", []string{"easy", "medium", "hard", "expert", "evil"}},
		{"strategy", []string{"naked-single", "hidden-single", "naked-pair", "naked-triple", "pointing-pair", "hidden-pair", "x-wing", "xy-wing", "hidden-triple", "w-wing", "swordfish", "naked-quad", "simple-coloring", "hidden-quad", "xyz-wing", "jellyfish", "bug-plus-one", "unique-rectangle", "unique-rectangle-2", "unique-rectangle-3", "unique-rectangle-4", "x-cycles", "xy-chain"}},
	} {
		for _, id := range domainIDs.ids {
			if _, err := RenderEnglishName(catalog, domainIDs.domain, id); err != nil {
				t.Errorf("%s %q: %v", domainIDs.domain, id, err)
			}
		}
	}
}

func TestRenderEnglishNameRejectsUnknownDomain(t *testing.T) {
	catalog, err := NewEnglishCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RenderEnglishName(catalog, "label", "easy"); err == nil {
		t.Fatal("RenderEnglishName accepted unsupported domain")
	}
}

func TestEnglishCatalogRejectsGenericStrategicHintMessages(t *testing.T) {
	catalog, err := NewEnglishCatalog()
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"hint.strategy.observe", "hint.strategy.compare", "hint.strategy.conclude"} {
		if _, err := catalog.Render(id); err == nil {
			t.Fatalf("Render(%q) accepted a retired generic strategic hint message", id)
		}
	}
}
