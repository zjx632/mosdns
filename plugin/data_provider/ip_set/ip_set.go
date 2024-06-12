/*
 * Copyright (C) 2020-2022, IrineSistiana
 *
 * This file is part of mosdns.
 *
 * mosdns is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * mosdns is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package ip_set

import (
	"bytes"
	"fmt"
	"github.com/IrineSistiana/mosdns/v5/coremain"
	"github.com/IrineSistiana/mosdns/v5/pkg/matcher/netlist"
	"github.com/IrineSistiana/mosdns/v5/plugin/data_provider"
	"net/netip"
	"os"
	"strings"
	"path/filepath"
)

const PluginType = "ip_set"

func init() {
	coremain.RegNewPluginFunc(PluginType, Init, func() any { return new(Args) })
}

func Init(bp *coremain.BP, args any) (any, error) {
	return NewIPSet(bp, args.(*Args))
}

type Args struct {
	IPs   []string `yaml:"ips"`
	Sets  []string `yaml:"sets"`
	Files []string `yaml:"files"`
}

var _ data_provider.IPMatcherProvider = (*IPSet)(nil)

type IPSet struct {
	mg []netlist.Matcher
}

func (d *IPSet) GetIPMatcher() netlist.Matcher {
	return MatcherGroup(d.mg)
}

func NewIPSet(bp *coremain.BP, args *Args) (*IPSet, error) {
	p := &IPSet{}

	l := netlist.NewList()
	if err := LoadFromIPsAndFiles(args.IPs, args.Files, l); err != nil {
		return nil, err
	}
	l.Sort()
	if l.Len() > 0 {
		p.mg = append(p.mg, l)
	}
	for _, tag := range args.Sets {
		provider, _ := bp.M().GetPlugin(tag).(data_provider.IPMatcherProvider)
		if provider == nil {
			return nil, fmt.Errorf("%s is not an IPMatcherProvider", tag)
		}
		p.mg = append(p.mg, provider.GetIPMatcher())
	}
	return p, nil
}

func parseNetipPrefix(s string) (netip.Prefix, error) {
	if strings.ContainsRune(s, '/') {
		return netip.ParsePrefix(s)
	}
	addr, err := netip.ParseAddr(s)
	if err != nil {
		return netip.Prefix{}, err
	}
	return addr.Prefix(addr.BitLen())
}

func LoadFromIPsAndFiles(ips []string, fs []string, l *netlist.List) error {
	if err := LoadFromIPs(ips, l); err != nil {
		return err
	}
	if err := LoadFromFiles(fs, l); err != nil {
		return err
	}
	return nil
}

func LoadFromIPs(ips []string, l *netlist.List) error {
	for i, s := range ips {
		p, err := parseNetipPrefix(s)
		if err != nil {
			return fmt.Errorf("invalid ip #%d %s, %w", i, s, err)
		}
		l.Append(p)
	}
	return nil
}

func LoadFromFiles(fs []string, l *netlist.List) error {
	for i, f := range fs {
		if err := LoadFromFile(f, l); err != nil {
			return fmt.Errorf("failed to load file #%d %s, %w", i, f, err)
		}
	}
	return nil
}

func LoadFromFile(f string, l *netlist.List) error {
	if len(f) > 0 {
		abspath, err := filepath.Abs(f)
		b, err := os.ReadFile(abspath)
		if err != nil {
			return err
		}
		if err := netlist.LoadFromReader(l, bytes.NewReader(b)); err != nil {
			return err
		}
	}
	return nil
}

type MatcherGroup []netlist.Matcher

func (mg MatcherGroup) Match(addr netip.Addr) bool {
	for _, m := range mg {
		if m.Match(addr) {
			return true
		}
	}
	return false
}
