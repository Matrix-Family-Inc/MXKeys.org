/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

import { useState } from 'react';

import { NotaryLookupForm, type NotaryLookupInput } from '@/features/notary-lookup';
import { useVerifyServer } from '@/features/notary-lookup/model/query';
import type { ServerKeys } from '@/features/notary-lookup/api/verify';

/**
 * NotaryLookupSection is a user-facing widget that exercises the
 * notary's own /_matrix/key/v2/query endpoint: the visitor types a
 * Matrix server_name and gets back the verify keys the notary has
 * cached or just freshly fetched.
 *
 * This is the only widget today that wires RHF + Zod + TanStack
 * Query + (in dev) MSW into production code. New forms follow the
 * same pattern: zod schema in features/<name>/model, mutation hook
 * in features/<name>/model, and a thin UI widget here.
 */
export function NotaryLookupSection() {
  const [lastQuery, setLastQuery] = useState<string | null>(null);
  const verify = useVerifyServer();

  const onSubmit = async (input: NotaryLookupInput) => {
    setLastQuery(input.server_name);
    verify.mutate({ serverName: input.server_name });
  };

  return (
    <section
      id="lookup"
      className="mx-auto w-full max-w-3xl px-6 py-16"
      aria-labelledby="lookup-heading"
    >
      <h2 id="lookup-heading" className="mb-2 text-2xl font-semibold">
        Look up a Matrix server
      </h2>
      <p className="mb-6 text-sm text-neutral-600">
        The notary returns the verify keys it has cached or freshly
        fetched for the given server_name.
      </p>

      <NotaryLookupForm onSubmit={onSubmit} busy={verify.isPending} />

      {verify.isError ? (
        <div
          role="alert"
          className="mt-4 rounded-md border border-red-300 bg-red-50 p-3 text-sm text-red-900"
        >
          <strong>Lookup failed:</strong> {verify.error.message}
        </div>
      ) : null}

      {verify.isSuccess ? (
        <LookupResult
          queried={lastQuery ?? ''}
          serverKeys={verify.data.server_keys}
        />
      ) : null}
    </section>
  );
}

interface LookupResultProps {
  queried: string;
  serverKeys: ServerKeys[];
}

function LookupResult({ queried, serverKeys }: LookupResultProps) {
  if (serverKeys.length === 0) {
    return (
      <div className="mt-4 rounded-md border border-neutral-300 bg-neutral-50 p-3 text-sm">
        The notary returned no keys for <code>{queried}</code>. The
        server may be unreachable or the cache is empty.
      </div>
    );
  }
  return (
    <ul className="mt-4 flex flex-col gap-3">
      {serverKeys.map((sk) => (
        <li
          key={`${sk.server_name}-${sk.valid_until_ts}`}
          className="rounded-md border border-neutral-300 bg-neutral-50 p-3"
        >
          <div className="flex items-baseline justify-between">
            <strong className="text-sm">{sk.server_name}</strong>
            <time
              dateTime={new Date(sk.valid_until_ts).toISOString()}
              className="text-xs text-neutral-600"
            >
              valid until {new Date(sk.valid_until_ts).toISOString()}
            </time>
          </div>
          <ul className="mt-2 flex flex-col gap-1">
            {Object.entries(sk.verify_keys).map(([keyID, v]) => (
              <li key={keyID} className="font-mono text-xs">
                <span className="text-neutral-500">{keyID}</span>:{' '}
                <span className="break-all">{v.key}</span>
              </li>
            ))}
          </ul>
        </li>
      ))}
    </ul>
  );
}
