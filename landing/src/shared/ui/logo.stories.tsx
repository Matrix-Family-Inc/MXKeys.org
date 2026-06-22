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

import type { Meta, StoryObj } from '@storybook/react';

import { Logo } from './logo';

const meta: Meta<typeof Logo> = {
  title: 'Shared / Logo',
  component: Logo,
};
export default meta;

type Story = StoryObj<typeof Logo>;

export const Default: Story = {};
