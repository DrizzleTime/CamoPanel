package docker

func publishedPorts(r CreateRequest) []Port {
	if len(r.Ports) > 0 {
		ports := make([]Port, 0, len(r.Ports))
		for _, p := range r.Ports {
			proto := p.Protocol
			if proto == "" {
				proto = "tcp"
			}
			ip := p.HostIP
			if ip == "" {
				ip = "127.0.0.1"
			}
			ports = append(ports, Port{IP: ip, PrivatePort: p.ContainerPort, PublicPort: p.HostPort, Type: proto})
		}
		return ports
	}
	if r.HostPort > 0 {
		return []Port{{IP: "127.0.0.1", PrivatePort: r.ContainerPort, PublicPort: r.HostPort, Type: "tcp"}}
	}
	return []Port{}
}
