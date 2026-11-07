package youtube

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// FriendlyError exposes localizeError to other packages for display purposes.
func FriendlyError(err error) error { return localizeError(err) }

// localizeError turns the most common upstream English errors into Turkish
// messages while keeping the original text attached. Classification code must
// keep using the raw error; this helper is only for user-facing text.
func localizeError(err error) error {
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "private video"):
		return fmt.Errorf("video ozel (private); erisim icin --cookies kullanin: %w", err)
	case strings.Contains(msg, "login required") || strings.Contains(msg, "sign in"):
		return fmt.Errorf("video icin oturum gerekiyor; --cookies ile giris yapin: %w", err)
	case strings.Contains(msg, "video not found") || strings.Contains(msg, "not found"):
		return fmt.Errorf("video bulunamadi: %w", err)
	case strings.Contains(msg, "rate limited") || strings.Contains(msg, "too many requests") || strings.Contains(msg, "429"):
		return fmt.Errorf("YouTube istekleri sinirladi; bir sure sonra tekrar deneyin: %w", err)
	case strings.Contains(msg, "sorry") || strings.Contains(msg, "unusual traffic"):
		return fmt.Errorf("YouTube bot dogrulamasi istedi; --cookies ile kendi oturumunuzu kullanin: %w", err)
	case strings.Contains(msg, "not playable in embed"):
		return fmt.Errorf("video gomulu oynatilamiyor: %w", err)
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Errorf("YouTube istegi zaman asimina ugradi: %w", err)
	}
	return err
}
