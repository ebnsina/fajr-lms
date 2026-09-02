import Video from '@lucide/svelte/icons/video';
import Headphones from '@lucide/svelte/icons/headphones';
import FileText from '@lucide/svelte/icons/file-text';
import FileType from '@lucide/svelte/icons/file-type';
import Link from '@lucide/svelte/icons/link';
import Radio from '@lucide/svelte/icons/radio';
import ListChecks from '@lucide/svelte/icons/list-checks';
import PenLine from '@lucide/svelte/icons/pen-line';
import type { Lesson } from '$lib/api';

/** One icon per lesson kind, so an outline can be scanned rather than read. */
const byKind = {
	video: Video,
	audio: Headphones,
	text: FileText,
	pdf: FileType,
	link: Link,
	live: Radio,
	quiz: ListChecks,
	assignment: PenLine
} as const;

export function lessonIcon(kind: Lesson['kind']) {
	return byKind[kind] ?? FileText;
}
