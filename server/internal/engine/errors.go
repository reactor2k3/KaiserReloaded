package engine

// RuleError ist ein Regelverstoß, den der Client dem Spieler direkt anzeigen darf.
// Code ist maschinenlesbar, Message der deutsche Text.
type RuleError struct {
	Code    string
	Message string
}

func (e *RuleError) Error() string { return e.Message }

// Bekannte Regelverstöße; Texte angelehnt an die Meldungen des Originals.
var (
	ErrInvalidAmount  = &RuleError{"invalid_amount", "Ungültige Menge."}
	ErrGrainExhausted = &RuleError{"grain_exhausted", "Vorräte erschöpft!"}
	ErrNotEnoughGrain = &RuleError{"not_enough_grain", "So viel Korn haben Sie nicht."}
	ErrNotEnoughLand  = &RuleError{"not_enough_land", "So viel Land haben Sie nicht."}
	ErrNoMoney        = &RuleError{"no_money", "Ihr Kreditrahmen ist ausgeschöpft."}
	ErrTooMuch        = &RuleError{"too_much", "Das ist zu viel!"}
	ErrTooLittle      = &RuleError{"too_little", "Das ist zu wenig!"}
	ErrAlreadyFed     = &RuleError{"already_fed", "Das Volk wurde dieses Jahr schon versorgt."}
	ErrBuildingLand   = &RuleError{"building_land", "Zu wenig Bauland!"}
	ErrTooManySoldier = &RuleError{"too_many_soldiers", "Zu viele Soldaten!"}
	ErrInvalidTax     = &RuleError{"invalid_tax", "Steuersätze müssen zwischen 0 und 99 % liegen."}
	ErrInvalidJustice = &RuleError{"invalid_justice", "Justiz muss zwischen 1 und 4 liegen."}
)
