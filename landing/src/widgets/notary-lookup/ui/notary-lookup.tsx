/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Wed Apr 22 2026 UTC
 * Status: Updated
 */

import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { NotaryLookupForm, type NotaryLookupInput } from '@/features/notary-lookup';
import { useVerifyServer } from '@/features/notary-lookup/model/query';
import type { ServerKeys } from '@/features/notary-lookup/api/verify';
import { ServerInfoPanel, useServerInfo } from '@/features/server-info';

const SUGGESTED_PORTS = [8448, 443];

/**
 * NotaryLookupSection renders the visitor-facing "look up a
 * Matrix server" experience. It exercises the notary's own
 * /_matrix/key/v2/query endpoint plus the optional server-info
 * enrichment endpoint; all user-visible strings are routed
 * through react-i18next so the same component works in every
 * shipped locale.
 */
export function NotaryLookupSection() {
  const { t } = useTranslation();
  const [lastQuery, setLastQuery] = useState<string | null>(null);
  const verify = useVerifyServer();
  const info = useServerInfo();

  const onSubmit = async (input: NotaryLookupInput) => {
    setLastQuery(input.server_name);
    verify.mutate({ serverName: input.server_name });
    info.mutate({ serverName: input.server_name });
  };

  const retryWithPort = (port: number) => {
    if (!lastQuery) return;
    const host = stripPort(lastQuery);
    const target = `${host}:${port}`;
    setLastQuery(target);
    verify.mutate({ serverName: target });
    info.mutate({ serverName: target });
  };

  const outcome = useMemo(() => classify(verify), [verify]);

  return (
    <section
      id="lookup"
      className="mx-auto w-full max-w-3xl px-6 py-16"
      aria-labelledby="lookup-heading"
    >
      <h2 id="lookup-heading" className="mb-2 text-2xl font-semibold text-text">
        {t('lookup.title')}
      </h2>
      <p className="mb-6 text-sm text-text-secondary">
        {t('lookup.description')}
      </p>

      <NotaryLookupForm onSubmit={onSubmit} busy={verify.isPending} />

      {outcome.kind === 'success' && lastQuery ? (
        <LookupResult queried={lastQuery} serverKeys={outcome.serverKeys} />
      ) : null}

      {outcome.kind === 'notfound' && lastQuery ? (
        <NotFoundHint
          queried={lastQuery}
          retryWithPort={retryWithPort}
          busy={verify.isPending}
        />
      ) : null}

      {outcome.kind === 'failed' ? (
        <div
          role="alert"
          className="mt-4 rounded-md border border-danger bg-bg-surface p-3 text-sm text-danger"
        >
          <strong>{t('lookup.failed')}</strong>{' '}
          <span className="text-text-secondary">{outcome.message}</span>
        </div>
      ) : null}

      {info.isSuccess && info.data ? <ServerInfoPanel info={info.data} /> : null}
    </section>
  );
}

interface LookupResultProps {
  queried: string;
  serverKeys: ServerKeys[];
}

function LookupResult({ queried, serverKeys }: LookupResultProps) {
  const { t } = useTranslation();
  return (
    <ul className="mt-4 flex flex-col gap-3">
      {serverKeys.map((sk) => (
        <li key={`${sk.server_name}-${sk.valid_until_ts}`} className="card">
          <div className="flex items-baseline justify-between">
            <strong className="text-sm text-text">{sk.server_name}</strong>
            <time
              dateTime={new Date(sk.valid_until_ts).toISOString()}
              className="text-xs text-text-muted"
            >
              {t('lookup.result.validUntil')}{' '}
              {new Date(sk.valid_until_ts).toISOString()}
            </time>
          </div>
          <ul className="mt-2 flex flex-col gap-1">
            {Object.entries(sk.verify_keys).map(([keyID, v]) => (
              <li key={keyID} className="font-mono text-xs">
                <span className="text-text-muted">{keyID}</span>:{' '}
                <span className="break-all text-text">{v.key}</span>
              </li>
            ))}
          </ul>
        </li>
      ))}
      <p className="text-xs text-text-muted">
        {t('lookup.result.footer', { server: queried })}
      </p>
    </ul>
  );
}

interface NotFoundHintProps {
  queried: string;
  retryWithPort: (port: number) => void;
  busy: boolean;
}

function NotFoundHint({ queried, retryWithPort, busy }: NotFoundHintProps) {
  const { t } = useTranslation();
  const hasPort = !queried.startsWith('[') && queried.includes(':');
  return (
    <div
      role="status"
      className="mt-4 rounded-md border border-border bg-bg-surface p-4 text-sm text-text-secondary"
    >
      <p className="text-text">
        {t('lookup.notfound.title', { server: queried })}
      </p>
      {hasPort ? (
        <p className="mt-2">{t('lookup.notfound.withPort')}</p>
      ) : (
        <>
          <p className="mt-2">{t('lookup.notfound.withoutPort')}</p>
          <div className="mt-3 flex flex-wrap gap-2">
            {SUGGESTED_PORTS.map((port) => (
              <button
                key={port}
                type="button"
                onClick={() => retryWithPort(port)}
                disabled={busy}
                className="btn btn-outline text-xs"
              >
                {t('lookup.notfound.tryPort')}{' '}
                <code>{stripPort(queried)}:{port}</code>
              </button>
            ))}
          </div>
        </>
      )}
    </div>
  );
}

type Outcome =
  | { kind: 'idle' }
  | { kind: 'success'; serverKeys: ServerKeys[] }
  | { kind: 'notfound' }
  | { kind: 'failed'; message: string };

function classify(verify: ReturnType<typeof useVerifyServer>): Outcome {
  if (verify.isSuccess) {
    const keys = verify.data.server_keys;
    if (keys.length === 0) return { kind: 'notfound' };
    return { kind: 'success', serverKeys: keys };
  }
  if (verify.isError) {
    const msg = verify.error.message.toLowerCase();
    const looksLikeReachability =
      msg.includes('not found') ||
      msg.includes('404') ||
      msg.includes('no known servers') ||
      msg.includes('resolve') ||
      msg.includes('refused') ||
      msg.includes('timeout') ||
      msg.includes('tls') ||
      msg.includes('unreachable') ||
      msg.includes('failed to fetch');
    if (looksLikeReachability) return { kind: 'notfound' };
    return { kind: 'failed', message: verify.error.message };
  }
  return { kind: 'idle' };
}

function stripPort(serverName: string): string {
  if (serverName.startsWith('[')) {
    const closingBracket = serverName.indexOf(']');
    if (closingBracket === -1) return serverName;
    return serverName.slice(0, closingBracket + 1);
  }
  const colon = serverName.lastIndexOf(':');
  if (colon === -1) return serverName;
  return serverName.slice(0, colon);
}
