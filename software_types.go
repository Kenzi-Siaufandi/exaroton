package exaroton

// SoftwareType is the slug of a software family (e.g. "vanilla", "papermc").
// The constants below cover the types currently exposed by the API; unknown
// values can still be passed as SoftwareType("...") so new types keep working.
type SoftwareType string

const (
	SoftwareTypeVanilla    SoftwareType = "vanilla"
	SoftwareTypePaperMC    SoftwareType = "papermc"
	SoftwareTypeSpigot     SoftwareType = "spigot"
	SoftwareTypePurpur     SoftwareType = "purpur"
	SoftwareTypeFolia      SoftwareType = "folia"
	SoftwareTypeFabric     SoftwareType = "fabric"
	SoftwareTypeForge      SoftwareType = "forge"
	SoftwareTypeNeoForge   SoftwareType = "neoforge"
	SoftwareTypeQuilt      SoftwareType = "quilt"
	SoftwareTypeArclight   SoftwareType = "arclight"
	SoftwareTypeGlowstone  SoftwareType = "glowstone"
	SoftwareTypeNukkit     SoftwareType = "nukkit"
	SoftwareTypeBungeeCord SoftwareType = "bungeecord"
	SoftwareTypeVelocity   SoftwareType = "velocity"
	SoftwareTypeWaterfall  SoftwareType = "waterfall"
	SoftwareTypeBedrock    SoftwareType = "bedrock"
	SoftwareTypePreview    SoftwareType = "preview"
	SoftwareTypeSnapshot   SoftwareType = "snapshot"
	SoftwareTypeHytale     SoftwareType = "hytale"
	SoftwareTypeModpacks   SoftwareType = "modpacks"
)

// SoftwareLauncher identifies the launcher a modpack or plugin belongs to.
type SoftwareLauncher string

const (
	LauncherCurseForge SoftwareLauncher = "CurseForge"
	LauncherModrinth   SoftwareLauncher = "Modrinth"
	LauncherFTB        SoftwareLauncher = "FTB"
	LauncherATL        SoftwareLauncher = "ATL"
	LauncherTechnic    SoftwareLauncher = "Technic"
)
