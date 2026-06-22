/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Mon 22 Jun 2026 00:50:40 UTC
 * Status: Updated
 */

import { afterEach, describe, expect, it } from 'vitest';
import { cleanup, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

import { NotaryLookupSection } from './notary-lookup';

afterEach(cleanup);

function renderWithQuery(ui: React.ReactElement) {
  const qc = new QueryClient({
    defaultOptions: {
      mutations: { retry: false },
      queries: { retry: false },
    },
  });
  return render(<QueryClientProvider client={qc}>{ui}</QueryClientProvider>);
}

describe('NotaryLookupSection', () => {
  it('queries the notary and renders the returned keys via MSW', async () => {
    renderWithQuery(<NotaryLookupSection />);

    // role=textbox disambiguates the input from section labels that
    // also contain "Matrix server" text.
    await userEvent.type(
      screen.getByRole('textbox', { name: /matrix server/i }),
      'matrix.example.org',
    );
    await userEvent.click(screen.getByRole('button', { name: /check notary keys/i }));

    // The widget renders the queried hostname in two places: the
    // server_name headline of each returned key card, and the
    // "Result for <host>" footer. getAllBy* tolerates both; the
    // assertion is satisfied as long as at least one match exists.
    await waitFor(() => {
      expect(screen.getAllByText(/matrix\.example\.org/).length).toBeGreaterThan(0);
    });
    expect(screen.getByText(/ed25519:auto/i)).toBeInTheDocument();
  });
});
