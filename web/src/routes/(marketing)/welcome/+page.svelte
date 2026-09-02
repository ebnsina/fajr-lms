<script lang="ts">
	import Check from '@lucide/svelte/icons/check';
	import X from '@lucide/svelte/icons/x';
	import Globe from '@lucide/svelte/icons/globe';
	import Wallet from '@lucide/svelte/icons/wallet';
	import WifiOff from '@lucide/svelte/icons/wifi-off';
	import ClipboardCheck from '@lucide/svelte/icons/clipboard-check';
	import Video from '@lucide/svelte/icons/video';
	import Users from '@lucide/svelte/icons/users';
	import Layout from '@lucide/svelte/icons/layout-template';
	import ShieldCheck from '@lucide/svelte/icons/shield-check';

	const reasons = [
		{
			icon: Wallet,
			title: 'The payments your families already use',
			body: 'bKash, SSLCommerz and plain bank transfer with a deposit slip somebody on staff approves. No card required to enroll a single learner.'
		},
		{
			icon: Globe,
			title: 'Arabic and Bengali as first-class text',
			body: 'Right to left and left to right render properly side by side, down to the punctuation and the numerals, on every screen.'
		},
		{
			icon: WifiOff,
			title: 'Built for a phone on a weak connection',
			body: 'Progress is merged, not overwritten, so a lesson watched on the bus still counts when the signal comes back.'
		},
		{
			icon: ClipboardCheck,
			title: 'The whole teaching week',
			body: 'Lessons, quizzes, assignments, grading, a gradebook that weighs it all, attendance that tells the guardians, and certificates anyone can verify.'
		},
		{
			icon: Video,
			title: 'Your live classes, your media',
			body: 'Paste a Meet or Zoom link today. Media is pluggable, so your own transcoder drops in without changing a single lesson.'
		},
		{
			icon: Users,
			title: 'One place, many schools',
			body: 'A teacher who works at three institutions signs in once. Every school keeps its own data, enforced by the database itself.'
		}
	];

	const steps = [
		{
			title: 'Open the school',
			body: 'A name, an address for your site, the script you teach in. About a minute, and no card.'
		},
		{
			title: 'Build a course',
			body: 'Sections and lessons, a pasted video link, a quiz with its answer key, an assignment with a due date.'
		},
		{
			title: 'Let the learners in',
			body: 'Invite them, or let them enroll and pay by bKash or bank transfer while a person on staff approves it.'
		},
		{
			title: 'Run the term',
			body: 'Grade what comes in, take the register, watch the gradebook fill, hand out certificates at the end.'
		}
	];

	const audiences = [
		{
			title: 'For a madrasah',
			body: 'Hifz cycles and the board syllabus side by side, guardians told the same morning a child is missing, fees collected without a card.',
			line: 'Attendance · guardian alerts · Arabic-first pages'
		},
		{
			title: 'For a school or college',
			body: 'A notice board the office can update itself, a weighted gradebook that survives a full term, and results parents can actually find.',
			line: 'Gradebook · notices · a public website'
		},
		{
			title: 'For a teacher on their own',
			body: 'One course, one price, one page that sells it. No institution behind you and no monthly fee until you have fifty learners.',
			line: 'Free to start · your own site · your own media'
		}
	];

	const comparison = [
		{ line: 'Enrollment paid by mobile wallet or bank slip', us: true, them: false },
		{ line: 'Arabic and Bengali rendered properly, not bolted on', us: true, them: false },
		{ line: 'A public website with a page builder included', us: true, them: false },
		{ line: 'Attendance that reaches the guardian', us: true, them: false },
		{ line: 'Every teaching feature on every plan', us: true, them: false },
		{ line: 'Runs on open-source parts you could host yourself', us: true, them: false }
	];

	const included = [
		'A public website with a page builder',
		'Templates for schools, colleges, madrasahs and universities',
		'Courses, modules and lessons',
		'Quizzes graded automatically or by hand',
		'Assignments with file hand-in',
		'A weighted gradebook',
		'Attendance with guardian alerts',
		'Certificates with public verification',
		'Live classes with recordings',
		'Notifications by SMS, email or webhook'
	];

	const faq = [
		{
			q: 'Do we need a card to start?',
			a: 'No. Opening a school is free, and the first fifty learners cost nothing. When you do pay, it is by bank transfer or bKash.'
		},
		{
			q: 'Can we keep our own video hosting?',
			a: 'Yes. Paste a YouTube or Vimeo link today; media is a plug, so your own transcoder or object store drops in later without touching a lesson.'
		},
		{
			q: 'Where does our data live?',
			a: 'On the Institution plan, in your country. Everything is Postgres, and every school is isolated by the database itself, not by application code.'
		},
		{
			q: 'What if we already have a website?',
			a: 'Keep it. The pages here are optional; plenty of schools use only the catalog page and link to it from what they already have.'
		}
	];
</script>

<svelte:head>
	<title>Fajr LMS</title>
	<meta
		name="description"
		content="A learning platform for schools and teachers in South Asia and the Gulf: local payments, Arabic and Bengali, and a whole teaching week in one place."
	/>
</svelte:head>

<section class="border-b border-line bg-raised px-6 py-20 sm:py-28">
	<div class="mx-auto max-w-3xl text-center">
		<p class="mb-3 text-sm font-medium tracking-wide uppercase text-brand-text">
			For madrasahs, schools, coaching centers and teachers
		</p>
		<h1 class="text-4xl font-semibold sm:text-5xl">Run the whole school year in one place</h1>
		<p class="mx-auto mt-4 max-w-2xl text-lg text-ink-soft">
			Fajr LMS teaches, grades, collects the fees and keeps the guardians informed — in Arabic,
			Bengali or English, on the phones your learners already own.
		</p>
		<div class="mt-8 flex flex-wrap justify-center gap-3">
			<a class="btn" href="/start">Open a school</a>
			<a class="btn btn-quiet" href="/pricing">See the pricing</a>
		</div>
		<p class="mt-4 text-sm text-ink-soft">Free while you set it up. No card to begin.</p>
	</div>
</section>

<section class="border-b border-line px-6 py-12">
	<dl class="mx-auto grid max-w-5xl gap-6 text-center sm:grid-cols-4">
		<div>
			<dt class="font-mono text-3xl font-medium tabular-nums">3</dt>
			<dd class="mt-1 mb-0 text-sm text-ink-soft">scripts rendered properly</dd>
		</div>
		<div>
			<dt class="font-mono text-3xl font-medium tabular-nums">4</dt>
			<dd class="mt-1 mb-0 text-sm text-ink-soft">ways to take a fee</dd>
		</div>
		<div>
			<dt class="font-mono text-3xl font-medium tabular-nums">1</dt>
			<dd class="mt-1 mb-0 text-sm text-ink-soft">minute to open a school</dd>
		</div>
		<div>
			<dt class="font-mono text-3xl font-medium tabular-nums">0</dt>
			<dd class="mt-1 mb-0 text-sm text-ink-soft">features behind a higher tier</dd>
		</div>
	</dl>
</section>

<section class="mx-auto max-w-5xl px-6 py-16">
	<h2 class="mb-2 text-2xl font-semibold">Why schools here choose it</h2>
	<p class="mb-8 max-w-2xl text-ink-soft">
		Most platforms are built for a card-paying learner on fiber. This one is not.
	</p>
	<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
		{#each reasons as reason (reason.title)}
			<article class="card">
				<reason.icon class="mb-3 text-brand-text" size={22} aria-hidden="true" />
				<h3 class="mb-1 font-medium">{reason.title}</h3>
				<p class="mb-0 text-sm text-ink-soft">{reason.body}</p>
			</article>
		{/each}
	</div>
</section>

<section class="border-y border-line bg-raised px-6 py-16">
	<div class="mx-auto max-w-5xl">
		<h2 class="mb-2 text-2xl font-semibold">From nothing to teaching, in four steps</h2>
		<p class="mb-8 max-w-2xl text-ink-soft">
			Each one is a screen, not a project. Nobody has to call us to get through them.
		</p>
		<ol class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
			{#each steps as step, index (step.title)}
				<li class="card">
					<span class="mb-3 block font-mono text-sm text-brand-text">
						{String(index + 1).padStart(2, '0')}
					</span>
					<h3 class="mb-1 font-medium">{step.title}</h3>
					<p class="mb-0 text-sm text-ink-soft">{step.body}</p>
				</li>
			{/each}
		</ol>
	</div>
</section>

<section class="mx-auto max-w-5xl px-6 py-16">
	<h2 class="mb-2 text-2xl font-semibold">Your website, without a web developer</h2>
	<p class="mb-8 max-w-2xl text-ink-soft">
		Every school gets a public site built from sections you fill in. Start from a template written
		for your kind of institution, in the language it teaches in.
	</p>
	<div class="grid gap-4 sm:grid-cols-2">
		<article class="card">
			<Layout class="mb-3 text-brand-text" size={22} aria-hidden="true" />
			<h3 class="mb-1 font-medium">Templates that fit</h3>
			<p class="mb-0 text-sm text-ink-soft">
				A school, a college, a madrasah or a university — for Bangladesh and for the Gulf. Notice
				boards, admission pages, the numbers that matter, and your catalog pulled straight from the
				courses you teach.
			</p>
		</article>
		<article class="card">
			<Globe class="mb-3 text-brand-text" size={22} aria-hidden="true" />
			<h3 class="mb-1 font-medium">Dressed for where you teach</h3>
			<p class="mb-0 text-sm text-ink-soft">
				The Gulf setting reads right to left and sets Arabic a size larger; the Bengal setting sets
				Bengali and runs a little tighter. One choice, and the whole site follows.
			</p>
		</article>
	</div>
</section>

<section class="border-y border-line bg-raised px-6 py-16">
	<div class="mx-auto max-w-5xl">
		<h2 class="mb-8 text-2xl font-semibold">Built for who you are</h2>
		<div class="grid gap-4 lg:grid-cols-3">
			{#each audiences as audience (audience.title)}
				<article class="card flex flex-col">
					<h3 class="mb-2 font-medium">{audience.title}</h3>
					<p class="mb-4 flex-1 text-sm text-ink-soft">{audience.body}</p>
					<p class="mb-0 font-mono text-xs text-ink-faint">{audience.line}</p>
				</article>
			{/each}
		</div>
	</div>
</section>

<section class="mx-auto max-w-3xl px-6 py-16">
	<h2 class="mb-2 text-2xl font-semibold">What the big platforms leave out</h2>
	<p class="mb-6 text-ink-soft">
		Not a knock on them — they were built for a different learner, in a different country.
	</p>
	<div class="overflow-x-auto">
		<table class="w-full border-collapse text-sm">
			<thead>
				<tr class="border-b border-line text-start">
					<th class="py-3 text-start font-medium">&nbsp;</th>
					<th class="w-24 py-3 font-medium">Fajr LMS</th>
					<th class="w-24 py-3 font-medium text-ink-soft">The usual</th>
				</tr>
			</thead>
			<tbody>
				{#each comparison as row (row.line)}
					<tr class="border-b border-line">
						<td class="py-3 pe-4">{row.line}</td>
						<td class="py-3 text-center">
							<Check class="mx-auto text-brand-text" size={18} aria-label="yes" />
						</td>
						<td class="py-3 text-center">
							<X class="mx-auto text-ink-faint" size={18} aria-label="no" />
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
</section>

<section class="border-y border-line bg-raised px-6 py-16">
	<div class="mx-auto grid max-w-5xl gap-10 lg:grid-cols-2">
		<div>
			<h2 class="mb-2 text-2xl font-semibold">Everything is included</h2>
			<p class="mb-4 text-ink-soft">
				There is no teaching feature behind a higher tier. What changes with the plan is how many
				learners you teach.
			</p>
			<a class="btn btn-quiet" href="/pricing">See the pricing</a>
		</div>
		<ul class="grid gap-3 sm:grid-cols-2">
			{#each included as item (item)}
				<li class="flex items-start gap-2 text-sm">
					<Check class="mt-0.5 shrink-0 text-brand-text" size={16} aria-hidden="true" />
					{item}
				</li>
			{/each}
		</ul>
	</div>
</section>

<section class="mx-auto max-w-5xl px-6 py-16">
	<div class="card flex flex-col gap-4 sm:flex-row sm:items-start">
		<ShieldCheck class="shrink-0 text-brand-text" size={26} aria-hidden="true" />
		<div>
			<h2 class="mb-2 text-lg font-medium">Your school's data is your school's</h2>
			<p class="mb-0 text-sm text-ink-soft">
				Every school is isolated in the database itself, not by application code that could forget.
				Nothing is sold on, nothing is trained on, and you can take everything out in full, at any
				time. On the Institution plan it stays in your country.
			</p>
		</div>
	</div>
</section>

<section class="border-t border-line px-6 py-16">
	<div class="mx-auto max-w-3xl">
		<h2 class="mb-6 text-2xl font-semibold">Questions we are asked</h2>
		<div class="flex flex-col gap-3">
			{#each faq as item (item.q)}
				<div class="card">
					<h3 class="mb-1 font-medium">{item.q}</h3>
					<p class="mb-0 text-sm text-ink-soft">{item.a}</p>
				</div>
			{/each}
		</div>
	</div>
</section>

<section class="mx-auto max-w-3xl px-6 py-20 text-center">
	<h2 class="mb-2 text-2xl font-semibold">Your school could be teaching by this evening</h2>
	<p class="mb-6 text-ink-soft">
		Opening a school takes a name and a phone number. You can invite a teacher straight after.
	</p>
	<div class="flex flex-wrap justify-center gap-3">
		<a class="btn" href="/start">Open a school</a>
		<a class="btn btn-quiet" href="/pricing">See the pricing</a>
	</div>
</section>
