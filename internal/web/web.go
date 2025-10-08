// Package web implements an HTMX-based web application mirroring the TUI functionality.
//
// # HTMX Web Application Implementation Plan
//
// # Architecture
//
// The web app replicates the five-view TUI workflow using server-side rendering
// with HTMX for dynamic updates. Each view corresponds to a template and handler:
//
//  1. Playlist List: Server-rendered table with hx-get for track preview
//  2. Track Preview: HTMX partial swap showing tracks + transfer button
//  3. Transfer Confirm: Modal confirmation with hx-post trigger
//  4. Progress Monitor: SSE (Server-Sent Events) streaming progress updates
//  5. Results Display: Final status with matched/failed tracks breakdown
//
// Core Components
//
//   - HTTP Server: net/http server with html/template rendering
//   - Service Integration: Uses same services.Service and tasks.PlaylistEngine as TUI
//   - Session Management: Cookie-based sessions for OAuth state and user tracking
//   - SSE Handler: Streams real-time progress during transfers
//
// Routes
//
//	GET  /                      → Playlist list view (requires auth)
//	GET  /auth/spotify          → OAuth initiation
//	GET  /auth/spotify/callback → OAuth completion
//	GET  /playlists/{id}/tracks → HTMX partial: track list
//	POST /transfer              → Start transfer, return SSE endpoint
//	GET  /transfer/{id}/stream  → SSE progress stream
//	GET  /transfer/{id}/result  → Final result view
//
// Templates
//
//   - base.html: Layout with navigation, auth status
//   - playlists.html: Table with hx-get on rows
//   - tracks.html: Partial template for track preview
//   - progress.html: SSE consumer with progress bar
//   - results.html: Success/failure breakdown
//
// # State Management
//
// Unlike the TUI's in-memory state, the web app persists state in:
//   - Session cookies: Authentication tokens, user ID
//   - MigrationJob records: Track transfer progress across requests
//   - In-memory channels: SSE connections for active transfers
//
// # Progress Streaming
//
// Transfer progress uses Server-Sent Events:
//  1. POST /transfer creates MigrationJob, returns job ID
//  2. Client opens SSE connection to /transfer/{id}/stream
//  3. Handler launches goroutine running PlaylistEngine.Run
//  4. Progress channel updates stream as SSE events
//  5. On completion, send "done" event with redirect URL
//
// Authentication Flow
//
//  1. User visits /, redirected to /auth/spotify if not authenticated
//  2. OAuth dance stores tokens in session
//  3. Session middleware validates tokens on protected routes
//  4. Expired tokens trigger reauthorization flow
//
// Dependencies
//
//   - html/template: Server-side rendering
//   - net/http: HTTP server and SSE
//   - gorilla/sessions or similar: Cookie management
//
// Implementation Tasks
//
//  1. HTTP server setup with route registration
//  2. Template structure with HTMX integration
//  3. Session middleware for auth state
//  4. Playlist list handler with service integration
//  5. Track preview handler (HTMX partial)
//  6. Transfer endpoint creating MigrationJob
//  7. SSE handler streaming progress updates
//  8. Result handler displaying MigrationJob outcome
//  9. OAuth handlers wrapping existing Spotify auth
//  10. Error handling and validation
//
// # Testing Strategy
//
// Use httptest:
//   - Mock services.Service for playlist/track data
//   - Mock tasks.PlaylistEngine for transfers
//   - Validate HTMX headers and response structure
//   - Test SSE stream formatting
package web
