package facts

// Default sensitive-path markers for Spike projection. This is a factual
// pattern list, not a policy engine; callers that need a different set can
// replace it in a later configurable API.
var sensitiveBasenames = []string{
	".env",
	".npmrc",
	".netrc",
	"id_rsa",
	"id_ed25519",
	"credentials",
	"secrets",
	"auth.json",
}

var sensitiveSuffixes = []string{
	".pem",
	".env",
	".env.local",
	".env.production",
}

var sensitiveSegments = []string{
	"/.ssh/",
	"/docker/config.json",
}
