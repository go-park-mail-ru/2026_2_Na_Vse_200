package validation

const (
	DefaultPageLimit = 100
	MaxPageLimit     = 100
	// MaxPageOffset ограничивает число строк, пропускаемых PostgreSQL.
	MaxPageOffset = 10_000
)

func ValidPagination(limit, offset int) bool {
	return limit >= 1 && limit <= MaxPageLimit && offset >= 0 && offset <= MaxPageOffset
}
