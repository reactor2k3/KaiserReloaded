package engine

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

const dataDir = "../../../data"

// newTestPlayer liefert einen Herrscher mit Startwerten und geernteter Ernte.
func newTestPlayer(t *testing.T, b *Balance, seed uint64) (*Player, *Rng) {
	t.Helper()
	rng := NewRng(seed)
	p := NewPlayer("Micha", 5, false, b, rng)
	Harvest(p, b, rng)
	return p, rng
}

func TestDataFilesLoad(t *testing.T) {
	b, err := LoadBalance(filepath.Join(dataDir, "balance.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(b, DefaultBalance()) {
		t.Errorf("balance.json weicht von DefaultBalance ab:\n%+v\n%+v", b, DefaultBalance())
	}
	w, err := LoadWorld(filepath.Join(dataDir, "world.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(w.Countries) != 9 || len(w.Routes) != 9 {
		t.Fatalf("erwartet 9 Länder und 9 Routenlisten, bekommen %d/%d", len(w.Countries), len(w.Routes))
	}
	if got := w.TitleName(9, true); got != "Kaiserin" {
		t.Errorf("TitleName(9, weiblich) = %q", got)
	}
	// Sachsen (5) grenzt laut Original an Preußen, Hessen, Bayern, Böhmen.
	if got := w.Neighbors(5); !reflect.DeepEqual(got, []int{1, 2, 3, 4}) {
		t.Errorf("Nachbarn Sachsen = %v", got)
	}
}

func TestDeterministic(t *testing.T) {
	b := DefaultBalance()
	run := func() *Player {
		p, rng := newTestPlayer(t, b, 42)
		if _, err := Feed(p, b, rng, p.Year.GrainNeed); err != nil {
			t.Fatal(err)
		}
		CollectIncome(p, rng)
		DrawTaxFactors(p, rng)
		CollectTaxes(p)
		EndYear(p, b, rng, 1700, 30, 0, 0.5)
		return p
	}
	if a, c := run(), run(); !reflect.DeepEqual(a, c) {
		t.Fatal("gleicher Seed ergibt unterschiedlichen Zustand")
	}
}

func TestHarvestBounds(t *testing.T) {
	for _, b := range []*Balance{DefaultBalance(), ClassicBalance()} {
		for seed := uint64(1); seed < 200; seed++ {
			p, _ := newTestPlayer(t, b, seed)
			y := p.Year
			if y.Weather < 1 || y.Weather > 5 || y.GrainPrice < 1 || y.LandPrice <= 0 || p.Grain < 0 {
				t.Fatalf("seed %d classic=%v: unplausible Ernte %+v grain=%d", seed, b.Classic, y, p.Grain)
			}
		}
	}
}

func TestGrainBuyLimit(t *testing.T) {
	b := DefaultBalance()
	p, _ := newTestPlayer(t, b, 7)
	p.Treasury = 1 << 30
	if err := BuyGrain(p, b, 3*p.Year.GrainNeed+1); !errors.Is(err, ErrGrainExhausted) {
		t.Fatalf("erwartet ErrGrainExhausted, bekommen %v", err)
	}
	if err := BuyGrain(p, b, p.Year.GrainNeed); err != nil {
		t.Fatal(err)
	}
}

func TestCreditLimitOnlyInRemake(t *testing.T) {
	remake, classic := DefaultBalance(), ClassicBalance()
	p, _ := newTestPlayer(t, remake, 3)
	p.Treasury = 0
	if err := BuyLand(p, remake, 1_000_000); !errors.Is(err, ErrNoMoney) {
		t.Fatalf("Remake: erwartet ErrNoMoney, bekommen %v", err)
	}
	q, _ := newTestPlayer(t, classic, 3)
	q.Treasury = 0
	if err := BuyLand(q, classic, 1_000_000); err != nil || q.Treasury >= 0 {
		t.Fatalf("Original: Kauf ohne Kassencheck erwartet, err=%v kasse=%d", err, q.Treasury)
	}
}

func TestSellLandKeepsOneHectare(t *testing.T) {
	b := DefaultBalance()
	p, _ := newTestPlayer(t, b, 9)
	if err := SellLand(p, p.Land); !errors.Is(err, ErrNotEnoughLand) {
		t.Fatalf("erwartet ErrNotEnoughLand, bekommen %v", err)
	}
	if err := SellLand(p, p.Land-1); err != nil || p.Land != 1 {
		t.Fatalf("Verkauf bis 1 ha: err=%v land=%d", err, p.Land)
	}
}

func TestFeedLimits(t *testing.T) {
	b := DefaultBalance()
	p, rng := newTestPlayer(t, b, 11)
	lo, hi := FeedLimits(p)
	if _, err := Feed(p, b, rng, hi+1); !errors.Is(err, ErrTooMuch) {
		t.Errorf("über 80 %%: %v", err)
	}
	if _, err := Feed(p, b, rng, lo-1); !errors.Is(err, ErrTooLittle) {
		t.Errorf("unter 20 %%: %v", err)
	}
	if _, err := Feed(p, b, rng, lo); err != nil {
		t.Fatal(err)
	}
	if _, err := Feed(p, b, rng, lo); !errors.Is(err, ErrAlreadyFed) {
		t.Errorf("zweimal versorgt: %v", err)
	}
	if p.Population < b.MinPopulation {
		t.Errorf("Bevölkerung unter Minimum: %d", p.Population)
	}
}

func TestJusticeOneEmigration(t *testing.T) {
	// Original: Justiz 1 vertreibt Einwohner; Remake: nicht.
	for _, tc := range []struct {
		b    *Balance
		want bool
	}{{ClassicBalance(), true}, {DefaultBalance(), false}} {
		p, rng := newTestPlayer(t, tc.b, 5)
		p.Justice = 1
		p.Population = 10000
		got := p.emigrate(tc.b, rng) > 0
		if got != tc.want {
			t.Errorf("classic=%v: Auswanderung bei Justiz 1 = %v, erwartet %v", tc.b.Classic, got, tc.want)
		}
	}
}

func TestBuildNeedsLand(t *testing.T) {
	b := DefaultBalance()
	p, rng := newTestPlayer(t, b, 13)
	p.Treasury = 1 << 30
	if err := Build(p, b, rng, Palace); !errors.Is(err, ErrBuildingLand) {
		t.Errorf("Palast unter 13 000 ha: %v", err)
	}
	for i := 0; i < 10; i++ { // 10 000 ha → bis 10 Märkte (Original: A/1000−1 < E sperrt)
		if err := Build(p, b, rng, Market); err != nil {
			t.Fatalf("Markt %d: %v", i+1, err)
		}
	}
	if err := Build(p, b, rng, Market); !errors.Is(err, ErrBuildingLand) {
		t.Errorf("11. Markt auf 10 000 ha: %v", err)
	}
	for i := 0; i < 3; i++ {
		if err := Build(p, b, rng, Mill); err != nil {
			t.Fatal(err)
		}
	}
	// Miliz = min(10/5×2, 3/3×2) = 2
	if p.Militia != 2 {
		t.Errorf("Miliz = %d, erwartet 2", p.Militia)
	}
}

func TestRecruitLimitAndMorale(t *testing.T) {
	b := DefaultBalance()
	p, _ := newTestPlayer(t, b, 17)
	DrawUnitPrices(p, NewRng(1))
	p.Treasury = 1 << 30
	p.Morale = 0.5
	if err := Recruit(p, b, Cavalry, false); err != nil {
		t.Fatal(err)
	}
	if p.Morale <= 0.5 || p.Morale >= 1 {
		t.Errorf("Moral nicht gemittelt: %v", p.Morale)
	}
	p.Population = 100
	if err := Recruit(p, b, Infantry, false); !errors.Is(err, ErrTooManySoldier) {
		t.Errorf("zu wenig Einwohner: %v", err)
	}
	if err := Recruit(p, b, Infantry, true); err != nil {
		t.Errorf("Söldner trotz wenig Einwohnern: %v", err)
	}
}

func TestTitleScoreAndPromotion(t *testing.T) {
	b := DefaultBalance()
	p, rng := newTestPlayer(t, b, 19)
	// Start: 3 (Kasse) + 1 (Land) + 1 (Einwohner) = 5 → Stufe 0
	if s := TitleScore(p); s != 0 {
		t.Fatalf("Startwertung %d, erwartet 0", s)
	}
	p.Markets, p.Mills, p.Treasury, p.Land, p.Population = 17, 17, 85000, 102000, 34000
	r := EndYear(p, b, rng, 1700, 0, 0, 0.5)
	if !r.Promoted || p.Title != 2 {
		t.Errorf("Aufstieg erwartet, Titel %d, Report %+v", p.Title, r)
	}
	p.Title, p.Palace = 7, 10
	if canPromote(p) {
		t.Error("König ohne vollständigen Palast")
	}
}

func TestBankruptcy(t *testing.T) {
	b := DefaultBalance()
	p, _ := newTestPlayer(t, b, 23)
	p.Markets, p.Treasury, p.Land = 5, -50000, 8000
	death := p.DeathYear
	Bankruptcy(p, b, 1710)
	// Schulden bleiben, wenn Gebäude und Landverkauf nicht reichen (wie im Original).
	if p.Markets != 0 || p.Land != 3000 || p.Treasury >= 0 || p.DeathYear != death-2 {
		t.Errorf("Bankrott falsch abgewickelt: %+v", p)
	}
}

func TestInterest(t *testing.T) {
	remake := DefaultBalance()
	p, rng := newTestPlayer(t, remake, 29)
	p.Treasury = 10000
	EndYear(p, remake, rng, 1700, 0, 0, 0.5)
	if p.Treasury != 10300 {
		t.Errorf("Remake-Guthabenzins: %d", p.Treasury)
	}
	p.Treasury = -10000
	EndYear(p, remake, rng, 1700, 0, 0, 0.5)
	if p.Treasury != -11000 {
		t.Errorf("Remake-Schuldzins: %d", p.Treasury)
	}
}

func TestRngRoundTrip(t *testing.T) {
	a := NewRng(99)
	a.Float()
	state, err := a.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var c Rng
	if err := c.UnmarshalBinary(state); err != nil {
		t.Fatal(err)
	}
	if a.Float() != c.Float() {
		t.Error("Zustand nach Wiederherstellung verschieden")
	}
}
