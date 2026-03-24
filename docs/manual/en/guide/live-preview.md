# Live Preview Server

The live preview server lets you see your documentation update in real-time as you write. This guide covers how to use the mdPress serve command and understand how live reloading works.

## Starting the Preview Server

Launch the live preview server with a single command:

```bash
mdpress serve
```

This starts a local web server and automatically opens your documentation in the default browser. The default URL is `http://localhost:5173`.

## Server Command Options

The `serve` command accepts several flags to customize the preview environment.

### Specifying the Port

Change the port if 5173 is already in use:

```bash
mdpress serve --port 3000
```

Your documentation will be available at `http://localhost:3000`.

### Specifying the Host

Allow access from other machines on your network:

```bash
mdpress serve --host 0.0.0.0
```

Now you can access the server from another computer using your machine's IP address, like `http://192.168.1.100:5173`.

### Auto-Opening the Browser

By default, mdPress opens the browser automatically. Disable this behavior:

```bash
mdpress serve --no-open
```

Useful when running on remote systems or in development environments where you manage the browser separately.

### Custom Output Directory

Serve from a specific output directory:

```bash
mdpress serve --output ./public
```

This is useful if you've already built your documentation and want to preview the built output instead of live-building from source files.

## Complete Server Configuration

Combine multiple flags for your specific setup:

```bash
mdpress serve --port 8080 --host 0.0.0.0 --no-open --output ./build
```

This starts a server on port 8080 accessible from any network interface, without opening a browser, serving pre-built files from the `./build` directory.

## File Watching and Auto-Rebuild

mdPress automatically watches your source files for changes and rebuilds affected content.

### Watched Files

The server monitors:

- Markdown files (`.md` files in your book directory)
- Configuration files (`book.yaml`, `LANGS.md`)
- Asset files (images, stylesheets, custom scripts)
- SUMMARY.md (table of contents)
- Language configuration files (in multi-language setups)

### Rebuild Process

When you save a file:

1. mdPress detects the change within 100ms
2. Relevant files are re-parsed and re-rendered
3. Updated content is prepared for serving
4. Live reload signal is sent to the browser
5. Browser updates the content atomically
6. Your view updates without losing scroll position or form state

### Incremental Building

For large documentation projects, mdPress uses incremental building:

- Only modified chapters are re-built
- Unchanged content reuses previous build results
- Shared assets are built once
- Search index is updated incrementally

This keeps rebuild times fast even for large projects.

## WebSocket-Based Live Reload

The live reload mechanism uses WebSocket connections for instant updates.

### How Live Reload Works

1. **Initial Connection**: Browser connects to the dev server on page load
2. **File Monitoring**: Server watches source files continuously
3. **Change Detection**: When a file changes, the server detects it
4. **WebSocket Message**: Server sends a reload message via WebSocket
5. **Content Update**: Browser receives the message and refreshes content
6. **Atomic Swap**: New content replaces old content atomically

### Atomic Swap Mechanism

The atomic swap ensures smooth updates:

- New HTML is prepared in memory before any page change
- CSS and JavaScript are loaded silently
- Old content is removed and new content is inserted in a single operation
- No intermediate states are visible to the user
- Scroll position is preserved when possible

This prevents flickering and maintains a smooth preview experience.

### Network Resilience

The WebSocket connection is resilient:

- Automatic reconnection if the connection drops
- Status indicator shows connection state (usually hidden when connected)
- If the server restarts, the browser automatically reconnects
- Graceful degradation if WebSocket is unavailable

## Development Workflow

### Typical Live Preview Session

1. Start the server:
   ```bash
   mdpress serve
   ```

2. Documentation automatically opens in your browser

3. Edit your Markdown files:
   ```bash
   vim docs/chapter-1.md
   ```

4. Save the file - preview updates instantly

5. Continue editing and previewing

### Multi-Monitor Setup

Use the live preview on one monitor while editing on another:

```bash
mdpress serve --port 5173 --host 0.0.0.0
```

View the documentation on a second monitor or device while editing in your text editor.

### Testing Across Devices

Access the preview from multiple devices on your network:

```bash
mdpress serve --host 0.0.0.0 --port 5173
```

Then navigate to `http://<your-machine-ip>:5173` from:
- Phone: `http://192.168.1.100:5173`
- Tablet: `http://192.168.1.100:5173`
- Other computer: `http://192.168.1.100:5173`

All devices see the same live updates in real-time.

## Performance Considerations

### Rebuild Times

For most documentation:
- Small changes: 50-100ms
- Full rebuild: 200-500ms
- Large projects (1000+ pages): 1-2 seconds

Times depend on your hardware and project complexity.

### Memory Usage

The live preview server:
- Keeps source files in memory for quick access
- Maintains the current build state
- Stores one level of history for undo/redo operations
- Typical memory usage: 100-500MB for most projects

For very large projects, you can restart the server to free memory.

### Disabling Auto-Rebuild

If automatic rebuilding impacts performance, disable it:

```bash
mdpress serve --no-watch
```

Then manually trigger rebuilds (though this feature may vary by configuration).

## Troubleshooting

### Port Already in Use

If port 5173 is in use, specify a different port:

```bash
mdpress serve --port 8080
```

Or find the process using the port:

```bash
lsof -i :5173        # macOS/Linux
netstat -ano | grep 5173  # Windows
```

### Changes Not Reflected

Ensure the file you edited is in the watched directory:
- Source Markdown files must be in your book directory
- Configuration must be in the root directory
- Assets must be in the assets folder

If still not working, try restarting the server:

```bash
# Kill the current server and restart
mdpress serve
```

### Slow Rebuild Times

For large projects, rebuild slowness can be reduced:

1. Check system resources (CPU, disk space)
2. Exclude large binary assets from the watch
3. Split documentation into multiple projects if possible
4. Use the pre-built output mode with `--output` flag

### Browser Not Connecting

If the browser tab shows "Waiting for connection":

1. Check that the server is running
2. Verify the correct URL (should match the port you specified)
3. Check if WebSocket is blocked by a firewall
4. Try accessing from a different browser
5. Restart both the server and browser

### Live Reload Not Working

WebSocket issues can prevent live reload:

1. Check browser console for connection errors
2. Ensure your proxy/firewall doesn't block WebSocket connections
3. Try accessing via IP address instead of localhost
4. Restart the development server

## Advanced Usage

### Serving Multiple Projects

Run separate servers on different ports:

```bash
# Terminal 1
cd project-a
mdpress serve --port 5173

# Terminal 2
cd project-b
mdpress serve --port 5174
```

Access both at `http://localhost:5173` and `http://localhost:5174`.

### Integration with IDEs

Most modern IDEs can run terminal commands. Set up your IDE to run `mdpress serve` and view the preview in a side panel or separate window.

### Environment Variables

Some configurations can be set via environment variables:

```bash
export MDPRESS_PORT=8080
export MDPRESS_HOST=0.0.0.0
mdpress serve
```

### Build Configuration

The live preview respects your `book.yaml` configuration for:
- Table of contents structure
- Theme settings
- Language configuration
- Custom scripts and styles
- Markdown extensions

Changes to `book.yaml` trigger a full rebuild of the preview.
