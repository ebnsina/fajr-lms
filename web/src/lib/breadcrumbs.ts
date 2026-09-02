export type Crumb = { label: string; href: string };

// Static labels for segments whose name doesn't come from loaded data.
const labels: Record<string, string> = {
	courses: 'Courses',
	grades: 'My grades',
	notifications: 'Notifications',
	grading: 'Grading',
	submissions: 'Submissions',
	members: 'Members',
	payments: 'Payments',
	settings: 'Settings',
	website: 'Website',
	classes: 'Live classes',
	certificates: 'Certificates',
	edit: 'Edit',
	tenant: 'Switch school',
	lessons: 'Lessons',
	gradebook: 'Gradebook',
	quiz: 'Quiz',
	assignment: 'Assignment'
};

const isID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

function humanize(segment: string): string {
	return segment.replace(/[-_]/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase());
}

/** Maps the current path to a breadcrumb trail, filling dynamic segments
    (a course slug, a lesson id) from whatever the route already loaded
    rather than fetching anything extra. */
export function breadcrumbs(pathname: string, data: Record<string, unknown>): Crumb[] {
	const segments = pathname.split('/').filter(Boolean);
	const crumbs: Crumb[] = [{ label: 'Home', href: '/' }];

	let href = '';
	for (let i = 0; i < segments.length; i++) {
		const segment = segments[i];
		href += `/${segment}`;
		let label = labels[segment] ?? humanize(decodeURIComponent(segment));

		if (segments[i - 1] === 'courses') {
			const outline = data.outline as { course?: { title?: string } } | undefined;
			const course = data.course as { title?: string } | undefined;
			label = outline?.course?.title ?? course?.title ?? label;
		} else if (segments[i - 1] === 'lessons') {
			const lesson = data.lesson as { title?: string } | undefined;
			const quiz = data.quiz as { title?: string } | undefined;
			const assignment = data.assignment as { title?: string } | undefined;
			label = lesson?.title ?? quiz?.title ?? assignment?.title ?? label;
		}

		// An id we could not name is noise. Skip the crumb rather than print a
		// mangled uuid, and let the next segment carry the trail.
		if (isID.test(segment) && label === humanize(decodeURIComponent(segment))) {
			continue;
		}

		crumbs.push({ label, href });
	}

	return crumbs;
}
