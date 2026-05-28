/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
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
