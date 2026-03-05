package cli

// Import providers to trigger registration via init()
import (
	_ "github.com/sudokatie/api-key-rotate/internal/providers/github"
	_ "github.com/sudokatie/api-key-rotate/internal/providers/railway"
	_ "github.com/sudokatie/api-key-rotate/internal/providers/vercel"
)
