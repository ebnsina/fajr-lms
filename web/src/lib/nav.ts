import Home from '@lucide/svelte/icons/house';
import Library from '@lucide/svelte/icons/library';
import GraduationCap from '@lucide/svelte/icons/graduation-cap';
import Award from '@lucide/svelte/icons/award';
import Bell from '@lucide/svelte/icons/bell';
import ClipboardCheck from '@lucide/svelte/icons/clipboard-check';
import Inbox from '@lucide/svelte/icons/inbox';
import Users from '@lucide/svelte/icons/users';
import Receipt from '@lucide/svelte/icons/receipt';
import Globe from '@lucide/svelte/icons/globe';
import School from '@lucide/svelte/icons/school';
import Megaphone from '@lucide/svelte/icons/megaphone';
import Route from '@lucide/svelte/icons/route';
import Trophy from '@lucide/svelte/icons/trophy';
import HeartHandshake from '@lucide/svelte/icons/heart-handshake';
import type { Component } from 'svelte';

export type NavItem = {
	label: string;
	href: string;
	icon: Component;
	/** Roles that may see it. Empty means everyone in the tenant. */
	roles?: string[];
	/** Shown only to somebody who is a guardian of a child at this school. */
	guardiansOnly?: boolean;
};

export type NavGroup = { title: string; items: NavItem[] };

const staff = ['owner', 'admin', 'instructor', 'assistant'];
const admin = ['owner', 'admin'];

// Grouped by what somebody came here to do, not by which table it reads.
const groups: NavGroup[] = [
	{
		title: 'Learning',
		items: [
			{ label: 'Home', href: '/', icon: Home },
			{ label: 'Courses', href: '/courses', icon: Library },
			{ label: 'Paths', href: '/paths', icon: Route },
			{ label: 'My grades', href: '/grades', icon: GraduationCap },
			{ label: 'Certificates', href: '/certificates', icon: Award },
			{ label: 'Standing', href: '/leaderboard', icon: Trophy },
			{ label: 'Notifications', href: '/notifications', icon: Bell },
			{ label: 'Your family', href: '/family', icon: HeartHandshake, guardiansOnly: true }
		]
	},
	{
		title: 'Teaching',
		items: [
			{
				label: 'Grading',
				href: '/grading',
				icon: ClipboardCheck,
				roles: staff
			},
			{ label: 'Submissions', href: '/submissions', icon: Inbox, roles: staff },
			{ label: 'Notices', href: '/notices', icon: Megaphone, roles: staff }
		]
	},
	{
		title: 'Administration',
		items: [
			{ label: 'The school', href: '/school', icon: School, roles: admin },
			{ label: 'Members', href: '/members', icon: Users, roles: admin },
			{ label: 'Payments', href: '/payments', icon: Receipt, roles: admin },
			{ label: 'Website', href: '/website', icon: Globe, roles: admin }
		]
	}
];

/** Hides whole groups a role cannot use, rather than showing dead entries. */
export function navFor(role: string | undefined, isGuardian = false): NavGroup[] {
	return groups
		.map((group) => ({
			...group,
			items: group.items.filter(
				(item) =>
					(!item.roles || (role && item.roles.includes(role))) &&
					(!item.guardiansOnly || isGuardian)
			)
		}))
		.filter((group) => group.items.length > 0);
}

/** The current page, matched on the longest href so children stay highlighted. */
export function isCurrent(href: string, pathname: string): boolean {
	if (href === '/') return pathname === '/';
	return pathname === href || pathname.startsWith(href + '/');
}
