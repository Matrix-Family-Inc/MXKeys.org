/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Wed Apr 22 2026 UTC
 * Status: Updated
 */

import type {
  ServerInfo,
  ServerInfoDns,
  ServerInfoReachability,
  ServerInfoWhois,
} from '../api/server-info';

/**
 * ServerInfoPanel renders the optional enrichment section under
 * the notary-lookup result: DNS / federation reachability /
 * WHOIS. Every sub-section is gated on the data actually
 * being present, because the backend returns `200` with whatever
 * succeeded within the request budget. An entirely empty
 * response collapses to `null`: the widget never renders a card
 * that says "nothing to show".
 */
export function ServerInfoPanel({ info }: { info: ServerInfo }) {
  if (!hasAnyData(info)) return null;

  return (
    <section className="mt-4 rounded-md border border-border bg-bg-surface p-4 text-sm text-text-secondary">
      <h3 className="mb-3 text-base font-semibold text-text">
        About <code>{info.server_name}</code>
      </h3>

      {info.reachability ? <ReachabilityRow reach={info.reachability} /> : null}
      {info.dns ? <DnsRows dns={info.dns} /> : null}
      {info.whois ? <WhoisRows who={info.whois} /> : null}
    </section>
  );
}

function hasAnyData(info: ServerInfo): boolean {
  return Boolean(info.dns || info.reachability || info.whois);
}

function Row({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="mt-2 grid grid-cols-[10rem_1fr] gap-x-4 gap-y-1 text-xs">
      <dt className="text-text-muted">{label}</dt>
      <dd className="text-text">{children}</dd>
    </div>
  );
}

function ReachabilityRow({ reach }: { reach: ServerInfoReachability }) {
  const status = reach.reachable ? 'reachable' : reach.error || 'unreachable';
  const colour = reach.reachable ? 'text-primary' : 'text-text-muted';
  return (
    <Row label="federation">
      <div>
        <span className={colour}>
          {status}
        </span>
        <span className="text-text-muted"> on port {reach.federation_port}</span>
        {reach.tls_version ? (
          <span className="text-text-muted">
            {' '} via {reach.tls_version}
            {reach.tls_sni_match === false ? ' (SNI mismatch)' : ''}
          </span>
        ) : null}
        {typeof reach.rtt_ms === 'number' && reach.rtt_ms > 0 ? (
          <span className="text-text-muted"> in {reach.rtt_ms} ms</span>
        ) : null}
      </div>
    </Row>
  );
}

function DnsRows({ dns }: { dns: ServerInfoDns }) {
  return (
    <>
      {dns.well_known_server ? (
        <Row label="well-known">
          <code className="font-mono text-xs">{dns.well_known_server}</code>
        </Row>
      ) : null}
      {dns.srv && dns.srv.length > 0 ? (
        <Row label="SRV">
          <ul className="flex flex-col gap-0.5">
            {dns.srv.map((s, i) => (
              <li key={`${s.target}-${s.port}-${i}`} className="font-mono text-xs">
                {s.target}:{s.port}
                <span className="text-text-muted">
                  {' '}(priority {s.priority}, weight {s.weight})
                </span>
              </li>
            ))}
          </ul>
        </Row>
      ) : null}
      {dns.resolved_host ? (
        <Row label="resolved">
          <code className="font-mono text-xs">
            {dns.resolved_host}
            {dns.resolved_port ? `:${dns.resolved_port}` : ''}
          </code>
        </Row>
      ) : null}
      {dns.a && dns.a.length > 0 ? (
        <Row label="IPv4">
          <span className="font-mono text-xs">{dns.a.join(', ')}</span>
        </Row>
      ) : null}
      {dns.aaaa && dns.aaaa.length > 0 ? (
        <Row label="IPv6">
          <span className="font-mono text-xs break-all">{dns.aaaa.join(', ')}</span>
        </Row>
      ) : null}
    </>
  );
}

function WhoisRows({ who }: { who: ServerInfoWhois }) {
  return (
    <>
      {who.registrar ? <Row label="registrar">{who.registrar}</Row> : null}
      {who.registered ? <Row label="registered">{who.registered}</Row> : null}
      {who.expires ? <Row label="expires">{who.expires}</Row> : null}
      {who.nameservers && who.nameservers.length > 0 ? (
        <Row label="nameservers">
          <span className="font-mono text-xs break-all">
            {who.nameservers.join(', ')}
          </span>
        </Row>
      ) : null}
    </>
  );
}
