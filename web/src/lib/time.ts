/** Relative while that still reads naturally, absolute once it stops. */
export function relativeTime(iso: string, locale?: string): string {
	const then = new Date(iso);
	const minutes = Math.round((then.getTime() - Date.now()) / 60_000);
	const rtf = new Intl.RelativeTimeFormat(locale, { numeric: 'auto' });
	if (Math.abs(minutes) < 60) return rtf.format(minutes, 'minute');
	if (Math.abs(minutes) < 60 * 24) return rtf.format(Math.round(minutes / 60), 'hour');
	if (Math.abs(minutes) < 60 * 24 * 7) return rtf.format(Math.round(minutes / 1440), 'day');
	return new Intl.DateTimeFormat(locale, { dateStyle: 'medium' }).format(then);
}
