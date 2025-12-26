package http

import (
	"fmt"
	"net/http"
	"path/filepath"
)

// ServeSwagger serves the Swagger UI documentation
func ServeSwagger(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeFile(w, r, filepath.Join("web", "templates", "swagger.html"))
}

// ServeOpenAPISpec serves the OpenAPI JSON specification
func ServeOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	http.ServeFile(w, r, filepath.Join("api", "openapi.json"))
}

// ServeMetricsPage wraps the Prometheus metrics with a styled HTML page
func ServeMetricsPage(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Prometheus Metrics - URL Shortener</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
        }

        .container {
            max-width: 1400px;
            margin: 0 auto;
        }

        .header {
            background: rgba(255, 255, 255, 0.95);
            backdrop-filter: blur(10px);
            border-radius: 16px;
            padding: 30px;
            margin-bottom: 20px;
            box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
        }

        .header h1 {
            font-size: 32px;
            font-weight: 700;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            margin-bottom: 10px;
        }

        .header p {
            color: #666;
            font-size: 16px;
        }

        .nav-buttons {
            display: flex;
            gap: 12px;
            margin-top: 20px;
        }

        .btn {
            padding: 12px 24px;
            border-radius: 8px;
            text-decoration: none;
            font-weight: 600;
            transition: all 0.3s;
            border: none;
            cursor: pointer;
            font-size: 14px;
        }

        .btn-primary {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
        }

        .btn-primary:hover {
            transform: translateY(-2px);
            box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);
        }

        .btn-secondary {
            background: white;
            color: #667eea;
            border: 2px solid #667eea;
        }

        .btn-secondary:hover {
            background: #f0f0f0;
        }

        .metrics-card {
            background: rgba(255, 255, 255, 0.95);
            backdrop-filter: blur(10px);
            border-radius: 16px;
            padding: 30px;
            box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
        }

        .metrics-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 20px;
            padding-bottom: 20px;
            border-bottom: 2px solid #f0f0f0;
        }

        .metrics-header h2 {
            font-size: 24px;
            color: #333;
        }

        .refresh-btn {
            padding: 8px 16px;
            background: #10b981;
            color: white;
            border: none;
            border-radius: 6px;
            cursor: pointer;
            font-weight: 600;
            transition: all 0.3s;
        }

        .refresh-btn:hover {
            background: #059669;
            transform: scale(1.05);
        }

        .metrics-content {
            background: #1e1e1e;
            border-radius: 8px;
            padding: 20px;
            overflow-x: auto;
            max-height: 70vh;
            font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
            font-size: 13px;
            line-height: 1.6;
            color: #d4d4d4;
        }

        .metrics-content pre {
            margin: 0;
            white-space: pre-wrap;
            word-wrap: break-word;
        }

        /* Syntax highlighting for metrics */
        .metric-comment {
            color: #6a9955;
        }

        .metric-name {
            color: #4ec9b0;
            font-weight: 600;
        }

        .metric-value {
            color: #b5cea8;
        }

        .metric-label {
            color: #ce9178;
        }

        .loading {
            text-align: center;
            padding: 40px;
            color: #666;
        }

        .spinner {
            border: 3px solid #f3f3f3;
            border-top: 3px solid #667eea;
            border-radius: 50%;
            width: 40px;
            height: 40px;
            animation: spin 1s linear infinite;
            margin: 0 auto 20px;
        }

        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }

        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 15px;
            margin-bottom: 20px;
        }

        .stat-card {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            padding: 20px;
            border-radius: 12px;
            color: white;
        }

        .stat-label {
            font-size: 12px;
            opacity: 0.9;
            margin-bottom: 8px;
        }

        .stat-value {
            font-size: 28px;
            font-weight: 700;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>📊 Prometheus Metrics</h1>
            <p>Real-time application metrics for monitoring and observability</p>
            <div class="nav-buttons">
                <a href="/" class="btn btn-primary">← Back to Home</a>
                <a href="/api/docs" class="btn btn-secondary">API Documentation</a>
                <button onclick="window.open('/metrics-raw', '_blank')" class="btn btn-secondary">Raw Metrics</button>
            </div>
        </div>

        <div class="stats-grid" id="statsGrid">
            <div class="stat-card">
                <div class="stat-label">Total Requests</div>
                <div class="stat-value" id="totalRequests">-</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Cache Hit Rate</div>
                <div class="stat-value" id="cacheHitRate">-</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Active URLs</div>
                <div class="stat-value" id="activeUrls">-</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Rate Limited</div>
                <div class="stat-value" id="rateLimited">-</div>
            </div>
        </div>

        <div class="metrics-card">
            <div class="metrics-header">
                <h2>Metrics Data</h2>
                <button class="refresh-btn" onclick="loadMetrics()">🔄 Refresh</button>
            </div>
            <div id="metricsContent" class="metrics-content">
                <div class="loading">
                    <div class="spinner"></div>
                    <p>Loading metrics...</p>
                </div>
            </div>
        </div>
    </div>

    <script>
        async function loadMetrics() {
            const content = document.getElementById('metricsContent');

            try {
                const response = await fetch('/metrics-raw');
                const text = await response.text();

                // Parse and highlight metrics
                const highlighted = highlightMetrics(text);
                content.innerHTML = '<pre>' + highlighted + '</pre>';

                // Update stats
                updateStats(text);
            } catch (error) {
                content.innerHTML = '<div class="loading"><p style="color: #ef4444;">Failed to load metrics: ' + error.message + '</p></div>';
            }
        }

        function highlightMetrics(text) {
            return text
                .split('\n')
                .map(line => {
                    if (line.startsWith('#')) {
                        return '<span class="metric-comment">' + escapeHtml(line) + '</span>';
                    } else if (line.includes('{')) {
                        const parts = line.split('{');
                        const name = '<span class="metric-name">' + escapeHtml(parts[0]) + '</span>';
                        const rest = parts[1] ? '{' + escapeHtml(parts[1]) : '';
                        return name + rest.replace(/="([^"]+)"/g, '=<span class="metric-label">"$1"</span>')
                                         .replace(/\s([\d.]+)$/, ' <span class="metric-value">$1</span>');
                    } else if (line.trim()) {
                        return line.replace(/^(\S+)\s+(.+)$/, '<span class="metric-name">$1</span> <span class="metric-value">$2</span>');
                    }
                    return line;
                })
                .join('\n');
        }

        function updateStats(text) {
            // Extract some key metrics
            const lines = text.split('\n');

            // Total requests
            const requestsLine = lines.find(l => l.startsWith('http_requests_total') && !l.startsWith('#'));
            if (requestsLine) {
                const match = requestsLine.match(/(\d+)$/);
                if (match) document.getElementById('totalRequests').textContent = match[1];
            }

            // Cache hits and misses
            const hitsLine = lines.find(l => l.startsWith('cache_hits_total') && !l.startsWith('#'));
            const missesLine = lines.find(l => l.startsWith('cache_misses_total') && !l.startsWith('#'));
            if (hitsLine && missesLine) {
                const hits = parseInt(hitsLine.match(/(\d+)$/)?.[1] || 0);
                const misses = parseInt(missesLine.match(/(\d+)$/)?.[1] || 0);
                const total = hits + misses;
                const rate = total > 0 ? ((hits / total) * 100).toFixed(1) : '0';
                document.getElementById('cacheHitRate').textContent = rate + '%';
            }

            // Active URLs
            const activeUrlsLine = lines.find(l => l.startsWith('active_urls') && !l.startsWith('#'));
            if (activeUrlsLine) {
                const match = activeUrlsLine.match(/(\d+)$/);
                if (match) document.getElementById('activeUrls').textContent = match[1];
            }

            // Rate limited
            const rateLimitedLine = lines.find(l => l.startsWith('rate_limited_requests_total') && !l.startsWith('#'));
            if (rateLimitedLine) {
                const match = rateLimitedLine.match(/(\d+)$/);
                if (match) document.getElementById('rateLimited').textContent = match[1];
            }
        }

        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }

        // Load metrics on page load
        loadMetrics();

        // Auto-refresh every 15 seconds
        setInterval(loadMetrics, 15000);
    </script>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, html)
}
