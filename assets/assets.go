package assets

import "embed"

//go:embed skills
var SkillsFS embed.FS

//go:embed templates
var TemplatesFS embed.FS

//go:embed web/static
var WebStaticFS embed.FS
