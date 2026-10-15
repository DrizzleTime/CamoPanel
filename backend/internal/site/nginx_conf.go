package site

import (
	"fmt"
	"strings"
)

// GenerateNginxConfig builds a production-grade Nginx server configuration for a site.
func GenerateNginxConfig(s *Site) string {
	allDomains := append([]string{s.PrimaryDomain}, s.Domains...)
	uniqueDomains := make([]string, 0, len(allDomains))
	seen := make(map[string]bool)
	for _, d := range allDomains {
		d = strings.TrimSpace(d)
		if d != "" && !seen[d] {
			seen[d] = true
			uniqueDomains = append(uniqueDomains, d)
		}
	}
	serverNames := strings.Join(uniqueDomains, " ")
	if serverNames == "" {
		serverNames = "_"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# CamoPanel Managed Site: %s (%s)\n", s.Name, s.ID))
	sb.WriteString(fmt.Sprintf("# Type: %s | SSL: %v\n\n", s.Type, s.SSLEnabled))

	if s.Status == "stopped" {
		sb.WriteString("server {\n")
		sb.WriteString("    listen 80;\n")
		if s.SSLEnabled {
			sb.WriteString("    listen 443 ssl;\n")
		}
		sb.WriteString(fmt.Sprintf("    server_name %s;\n", serverNames))
		sb.WriteString("    return 503 \"Site temporarily disabled by CamoPanel\";\n")
		sb.WriteString("}\n")
		return sb.String()
	}

	// Port 80 HTTP server block
	sb.WriteString("server {\n")
	sb.WriteString("    listen 80;\n")
	sb.WriteString(fmt.Sprintf("    server_name %s;\n\n", serverNames))

	// ACME challenge location for Let's Encrypt
	sb.WriteString("    # Let's Encrypt ACME HTTP-01 Challenge\n")
	sb.WriteString("    location /.well-known/acme-challenge/ {\n")
	sb.WriteString("        root /var/www/acme;\n")
	sb.WriteString("        try_files $uri =404;\n")
	sb.WriteString("    }\n\n")

	if s.SSLEnabled && s.ForceHTTPS {
		sb.WriteString("    # Force HTTPS Redirect\n")
		sb.WriteString("    location / {\n")
		sb.WriteString("        return 301 https://$host$request_uri;\n")
		sb.WriteString("    }\n")
		sb.WriteString("}\n\n")
	} else {
		writeSiteDirectives(&sb, s)
		sb.WriteString("}\n\n")
	}

	// Port 443 HTTPS server block
	if s.SSLEnabled {
		sb.WriteString("server {\n")
		sb.WriteString("    listen 443 ssl;\n")
		sb.WriteString("    http2 on;\n")
		sb.WriteString(fmt.Sprintf("    server_name %s;\n\n", serverNames))

		certDir := s.ID
		if s.CertID != "" {
			certDir = "certs/" + s.CertID
		}
		sb.WriteString(fmt.Sprintf("    ssl_certificate /etc/nginx/ssl/%s/fullchain.pem;\n", certDir))
		sb.WriteString(fmt.Sprintf("    ssl_certificate_key /etc/nginx/ssl/%s/privkey.pem;\n", certDir))
		sb.WriteString("    ssl_protocols TLSv1.2 TLSv1.3;\n")
		sb.WriteString("    ssl_ciphers HIGH:!aNULL:!MD5;\n")
		sb.WriteString("    ssl_prefer_server_ciphers on;\n")
		sb.WriteString("    ssl_session_cache shared:SSL:10m;\n")
		sb.WriteString("    ssl_session_timeout 1d;\n\n")

		sb.WriteString("    # Let's Encrypt ACME HTTP-01 Challenge\n")
		sb.WriteString("    location /.well-known/acme-challenge/ {\n")
		sb.WriteString("        root /var/www/acme;\n")
		sb.WriteString("        try_files $uri =404;\n")
		sb.WriteString("    }\n\n")

		writeSiteDirectives(&sb, s)
		sb.WriteString("}\n")
	}

	return sb.String()
}

func writeSiteDirectives(sb *strings.Builder, s *Site) {
	if s.Type == "static" {
		rootPath := fmt.Sprintf("/var/www/sites/%s", s.ID)
		if s.RootPath != "" {
			rootPath = s.RootPath
		}
		sb.WriteString(fmt.Sprintf("    root %s;\n", rootPath))
		sb.WriteString("    index index.html index.htm;\n\n")
		sb.WriteString("    location / {\n")
		sb.WriteString("        try_files $uri $uri/ /index.html =404;\n")
		sb.WriteString("    }\n\n")
		sb.WriteString("    location ~* \\.(?:css|js|jpg|jpeg|gif|png|ico|cur|gz|svg|svgz|mp4|ogg|ogv|webm|htc|woff2|woff)$ {\n")
		sb.WriteString("        expires 7d;\n")
		sb.WriteString("        access_log off;\n")
		sb.WriteString("        add_header Cache-Control \"public\";\n")
		sb.WriteString("    }\n")
	} else {
		// Reverse proxy
		target := s.ProxyTarget
		if target == "" {
			target = "http://127.0.0.1:8080"
		}
		sb.WriteString("    location / {\n")
		sb.WriteString(fmt.Sprintf("        proxy_pass %s;\n", target))
		sb.WriteString("        proxy_set_header Host $host;\n")
		sb.WriteString("        proxy_set_header X-Real-IP $remote_addr;\n")
		sb.WriteString("        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
		sb.WriteString("        proxy_set_header X-Forwarded-Proto $scheme;\n")
		sb.WriteString("        proxy_http_version 1.1;\n")
		sb.WriteString("        proxy_set_header Upgrade $http_upgrade;\n")
		sb.WriteString("        proxy_set_header Connection \"upgrade\";\n")
		sb.WriteString("        proxy_connect_timeout 60s;\n")
		sb.WriteString("        proxy_send_timeout 60s;\n")
		sb.WriteString("        proxy_read_timeout 60s;\n")
		sb.WriteString("    }\n")
	}

	for _, redirect := range s.Redirects {
		code := redirect.Code
		if code == 0 {
			code = 302
		}
		sb.WriteString(fmt.Sprintf("    location = %s {\n        return %d %s;\n    }\n", redirect.From, code, redirect.To))
	}
	for _, loc := range s.Locations {
		sb.WriteString(fmt.Sprintf("    location %s {\n        proxy_pass %s;\n    }\n", loc.Path, loc.Proxy))
	}
	if s.BasicAuthUser != "" {
		file := s.BasicAuthFile
		if file == "" {
			file = fmt.Sprintf("/etc/nginx/auth/%s.htpasswd", s.ID)
		}
		sb.WriteString("    auth_basic \"restricted\";\n")
		sb.WriteString(fmt.Sprintf("    auth_basic_user_file %s;\n", file))
	}
	if s.Hotlink {
		sb.WriteString("    valid_referers none blocked server_names;\n")
		sb.WriteString("    if ($invalid_referer) { return 403; }\n")
	}
	if s.RateLimit > 0 {
		sb.WriteString(fmt.Sprintf("    limit_req zone=camo_site burst=%d nodelay;\n", s.RateLimit))
	}
	sb.WriteString(fmt.Sprintf("    access_log /var/log/nginx/sites/%s.access.log;\n", s.ID))
	sb.WriteString(fmt.Sprintf("    error_log /var/log/nginx/sites/%s.error.log;\n", s.ID))

	if s.CustomConfig != "" {
		sb.WriteString("\n    # Custom Directives\n")
		for _, line := range strings.Split(s.CustomConfig, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				sb.WriteString(fmt.Sprintf("    %s\n", trimmed))
			}
		}
	}
}

// ValidateSnippet rejects config that leaves the server block or reads host files.
func ValidateSnippet(snippet string) error {
	lower := strings.ToLower(snippet)
	if strings.Contains(snippet, "}") || strings.Contains(lower, "include") || strings.Contains(snippet, "/etc/") || strings.Contains(snippet, "..") {
		return fmt.Errorf("自定义片段不能跳出 server 块或引用主机文件")
	}
	return nil
}
