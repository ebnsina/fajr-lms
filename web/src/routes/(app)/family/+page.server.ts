import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { api } from '$lib/server/api';

type Child = {
	student_id: string;
	relation: string;
	full_name: string;
	section_name: string | null;
	class_name: string | null;
};

type Hifz = {
	ayahs_memorised: number;
	lessons: number;
	total_ayahs: number;
	entries: {
		hifz_entry: {
			id: string;
			on_date: string;
			kind: string;
			from_surah: number;
			from_ayah: number;
			to_surah: number;
			to_ayah: number;
			quality: string;
			mistakes: number;
		};
		from_name: string;
		to_name: string;
		teacher_name: string | null;
	}[];
};

export const load: PageServerLoad = async ({ locals, parent, fetch }) => {
	if (!locals.token) redirect(303, '/login');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');

	const scoped = { token: locals.token, tenant: session.tenant.slug, fetch };
	const { children } = await api<{ children: Child[] }>('/v1/children', scoped);

	// Each child's hifz, where the school keeps one. A child with none simply
	// has nothing to show, which is not an error.
	const withHifz = await Promise.all(
		(children ?? []).map(async (child) => {
			try {
				const hifz = await api<Hifz>(`/v1/hifz/students/${child.student_id}?limit=8`, scoped);
				return { ...child, hifz };
			} catch {
				return { ...child, hifz: null as Hifz | null };
			}
		})
	);
	return { children: withHifz };
};
