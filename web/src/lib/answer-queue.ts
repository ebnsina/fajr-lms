import { browser } from '$app/environment';

export type QueuedAnswer = {
	attempt_id: string;
	question_id: string;
	option_ids: string[];
	text: string;
};

const KEY = 'fajr.answers';

// Answers written while the connection was gone. Keyed by attempt and question,
// so the last thing a learner chose is what gets sent, not every keystroke.
function read(): QueuedAnswer[] {
	if (!browser) return [];
	try {
		const raw = localStorage.getItem(KEY);
		return raw ? (JSON.parse(raw) as QueuedAnswer[]) : [];
	} catch {
		return [];
	}
}

function write(rows: QueuedAnswer[]) {
	try {
		localStorage.setItem(KEY, JSON.stringify(rows));
	} catch {
		// A full or blocked store is not worth failing the page over.
	}
}

export function queued(attemptID: string): QueuedAnswer[] {
	return read().filter((row) => row.attempt_id === attemptID);
}

export function hold(answer: QueuedAnswer) {
	const rows = read().filter(
		(row) => !(row.attempt_id === answer.attempt_id && row.question_id === answer.question_id)
	);
	rows.push(answer);
	write(rows);
}

function drop(answer: QueuedAnswer) {
	write(
		read().filter(
			(row) => !(row.attempt_id === answer.attempt_id && row.question_id === answer.question_id)
		)
	);
}

// flush sends what is waiting, keeping anything that still will not go. It
// returns how many are left so the page can say so.
export async function flush(action: string): Promise<number> {
	if (!browser) return 0;
	for (const answer of read()) {
		const body = new FormData();
		body.set('attempt_id', answer.attempt_id);
		body.set('question_id', answer.question_id);
		body.set('text', answer.text);
		for (const id of answer.option_ids) body.append('option_ids', id);

		try {
			const response = await fetch(action, { method: 'POST', body });
			// A refusal from the server will not become an acceptance later.
			if (response.ok || (response.status >= 400 && response.status < 500)) drop(answer);
		} catch {
			return read().length;
		}
	}
	return read().length;
}
