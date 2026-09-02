/** Shapes the pages consume. Only the fields the UI actually renders. */
export type Lesson = {
	id: string;
	module_id: string;
	title: string;
	kind: 'video' | 'audio' | 'text' | 'pdf' | 'link' | 'live' | 'quiz' | 'assignment';
	body: string;
	dir: 'auto' | 'ltr' | 'rtl';
	duration_s: number;
	is_preview: boolean;
	status: 'draft' | 'published' | 'archived';
	position: number;
	media_id: string | null;
};

export type Module = { id: string; title: string; position: number; lessons: Lesson[] };

export type Course = {
	id: string;
	slug: string;
	title: string;
	summary: string;
	dir: 'auto' | 'ltr' | 'rtl';
	status: string;
	price_minor: number;
	currency: string;
};

export type Outline = { course: Course; modules: Module[] };

export type LessonProgress = {
	lesson_id: string;
	state: 'not_started' | 'in_progress' | 'completed';
	position_s: number;
};

export type CourseProgress = {
	enrollment: { id: string; status: string };
	lessons: LessonProgress[];
	lessons_total: number;
	lessons_done: number;
	percent_complete: number;
};

export type Playback = {
	kind: 'embed' | 'hls' | 'file' | 'upload' | 'not_ready';
	url: string;
	expires_at?: string;
};

/** Direction the API stored, narrowed to what the dir attribute accepts. */
export function dirOf(value: string | undefined): 'auto' | 'ltr' | 'rtl' {
	return value === 'ltr' || value === 'rtl' ? value : 'auto';
}

/** Renders a length the way a learner reads it, not as raw seconds. */
export function duration(seconds: number, locale = 'en'): string {
	if (!seconds) return '';
	const minutes = Math.round(seconds / 60);
	if (minutes < 60) {
		return new Intl.NumberFormat(locale).format(minutes) + ' min';
	}
	const hours = Math.floor(minutes / 60);
	const rest = minutes % 60;
	const fmt = new Intl.NumberFormat(locale);
	return rest ? `${fmt.format(hours)} h ${fmt.format(rest)} min` : `${fmt.format(hours)} h`;
}

/** Joins the small facts under a title, skipping the ones that are absent. */
export function meta(...parts: (string | number | false | null | undefined)[]): string {
	return parts.filter(Boolean).join(' · ');
}

export function money(minor: number, currency: string, locale = 'en'): string {
	return new Intl.NumberFormat(locale, { style: 'currency', currency }).format(minor / 100);
}
