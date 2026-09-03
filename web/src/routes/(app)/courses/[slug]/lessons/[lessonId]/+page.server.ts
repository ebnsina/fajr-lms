import { error, fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import { api, ApiFailure } from '$lib/server/api';
import type { CourseProgress, Lesson, Outline, Playback } from '$lib/api';

type ScormLesson = {
	package: { title: string; entry_href: string; version: string; mastery: number | null };
	state: {
		lesson_status: string;
		suspend_data: string;
		location: string;
		total_time_s: number;
		score_raw: string | number | null;
	};
};

export const load: PageServerLoad = async ({ params, locals, parent, fetch }) => {
	if (!locals.token) redirect(303, '/login');
	const { session } = await parent();
	if (!session?.tenant) redirect(303, '/tenant');

	const scoped = { token: locals.token, tenant: session.tenant.slug, fetch };
	let outline: Outline;
	try {
		outline = await api<Outline>(`/v1/courses/${params.slug}`, scoped);
	} catch (failure) {
		if (failure instanceof ApiFailure && failure.status === 404) error(404, 'Course not found.');
		throw failure;
	}

	const ordered: Lesson[] = outline.modules.flatMap((module) => module.lessons);
	const index = ordered.findIndex((lesson) => lesson.id === params.lessonId);
	if (index < 0) error(404, 'That lesson is not part of this course.');

	let progress: CourseProgress | null = null;
	try {
		progress = await api<CourseProgress>(`/v1/courses/${outline.course.id}/progress`, scoped);
	} catch {
		progress = null;
	}

	// The playback URL expires, so it is fetched per view rather than cached.
	const lesson = ordered[index];
	let playback: Playback | null = null;
	if (lesson.media_id) {
		try {
			playback = await api<Playback>(`/v1/media/${lesson.media_id}/playback`, scoped);
		} catch {
			playback = null;
		}
	}

	// A lesson may be a package built elsewhere; most are not.
	let scorm: ScormLesson | null = null;
	try {
		scorm = await api<ScormLesson>(`/v1/lessons/${lesson.id}/scorm`, scoped);
	} catch {
		scorm = null;
	}

	const mine = progress?.lessons.find((row) => row.lesson_id === lesson.id);
	return {
		course: outline.course,
		lesson,
		playback,
		scorm,
		enrolled: progress !== null,
		state: mine?.state ?? null,
		resumeAt: mine?.position_s ?? 0,
		previous: index > 0 ? ordered[index - 1] : null,
		next: index + 1 < ordered.length ? ordered[index + 1] : null
	};
};

export const actions: Actions = {
	// Progress is reported through the server so the token stays in the cookie.
	progress: async ({ params, request, locals, cookies, fetch }) => {
		const tenant = cookies.get('fajr_tenant');
		if (!locals.token || !tenant) redirect(303, '/login');

		const form = await request.formData();
		const position = Number(form.get('position_s') ?? 0);
		const completed = form.get('completed') === 'true';

		try {
			await api(`/v1/lessons/${params.lessonId}/progress`, {
				method: 'PUT',
				token: locals.token,
				tenant,
				body: {
					position_s: Number.isFinite(position) ? Math.max(0, Math.trunc(position)) : 0,
					completed
				},
				fetch
			});
		} catch (failure) {
			if (failure instanceof ApiFailure)
				return fail(failure.status, { message: failure.error.message });
			throw failure;
		}
		return { saved: true, completed };
	}
};
