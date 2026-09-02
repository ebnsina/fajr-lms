import type { SiteBlock } from '$lib/types.site';

export type TemplatePage = {
	slug: string;
	title: string;
	nav_label: string;
	nav_order: number;
	description: string;
	blocks: SiteBlock[];
};

export type SiteTemplate = {
	id: string;
	name: string;
	kind: 'school' | 'college' | 'madrasah' | 'university';
	region: 'bengal' | 'gulf';
	theme: 'bengal' | 'gulf' | 'plain';
	summary: string;
	pages: TemplatePage[];
};

/** Every template is a starting point, not a cage: the copy is written to be
    replaced, and the sections are the ones these institutions actually publish. */
export const templates: SiteTemplate[] = [
	{
		id: 'bengal-school',
		name: 'Bengal school',
		kind: 'school',
		region: 'bengal',
		theme: 'bengal',
		summary: 'A notice board first, results and admissions close behind.',
		pages: [
			{
				slug: '',
				title: 'Your school',
				nav_label: '',
				nav_order: 0,
				description: 'A school in Bangladesh teaching from class six to the SSC.',
				blocks: [
					{
						type: 'hero',
						heading: 'Teaching this town since 1994',
						text: 'From class six to the SSC, with the results to show for it.',
						cta_label: 'Admissions',
						cta_href: '/admissions'
					},
					{
						type: 'notices',
						heading: 'Notice board',
						items: [
							{ title: 'Half-yearly results published', text: '12 June' },
							{ title: 'Class six admission forms open', text: '3 June' },
							{ title: 'School closed for Eid-ul-Azha', text: '28 May' }
						]
					},
					{
						type: 'stats',
						items: [
							{ title: '1,240', text: 'students' },
							{ title: '48', text: 'teachers' },
							{ title: '96%', text: 'SSC pass rate' },
							{ title: '1994', text: 'founded' }
						]
					},
					{ type: 'courses', heading: 'What we teach online', limit: 6 },
					{
						type: 'features',
						heading: 'Why parents choose us',
						items: [
							{ title: 'Small sittings', text: 'Thirty to a class, never more.' },
							{ title: 'Guardians are told', text: 'An absence reaches you the same morning.' },
							{ title: 'Fees you can plan', text: 'Monthly, by bKash or a bank slip.' }
						]
					},
					{
						type: 'cta',
						heading: 'Come and see the school',
						text: 'The office is open from 9 to 2, Sunday to Thursday.',
						cta_label: 'Create an account',
						cta_href: '/login'
					}
				]
			},
			{
				slug: 'admissions',
				title: 'Admissions',
				nav_label: 'Admissions',
				nav_order: 1,
				description: 'How to apply, what it costs, and when the forms close.',
				blocks: [
					{
						type: 'richtext',
						heading: 'Admissions',
						text: 'Forms for the coming year open in June and close at the end of July.\n\nBring the birth certificate, two photographs and the last report card. The admission test is held in the first week of August.'
					},
					{
						type: 'faq',
						heading: 'Questions parents ask',
						items: [
							{
								title: 'What does a month cost?',
								text: 'The monthly fee is 800 taka; the admission fee is 2,000 taka once.'
							},
							{
								title: 'Is there a scholarship?',
								text: 'Yes, for the top ten in the admission test, and for families in need.'
							},
							{
								title: 'How do we pay?',
								text: 'bKash or a bank transfer. Upload the slip and the office confirms it.'
							}
						]
					}
				]
			}
		]
	},
	{
		id: 'bengal-madrasah',
		name: 'Bengal madrasah',
		kind: 'madrasah',
		region: 'bengal',
		theme: 'bengal',
		summary: 'Hifz and the Dakhil syllabus side by side.',
		pages: [
			{
				slug: '',
				title: 'Your madrasah',
				nav_label: '',
				nav_order: 0,
				description: 'Hifz, Alim and the general syllabus under one roof.',
				blocks: [
					{
						type: 'hero',
						heading: 'The Qur’an first, and the syllabus with it',
						text: 'Hifz in the morning, Dakhil subjects after — so no child has to choose.',
						cta_label: 'See what we teach',
						cta_href: '/courses'
					},
					{
						type: 'features',
						heading: 'Three ways to study here',
						items: [
							{ title: 'Hifz', text: 'A five-year cycle with daily revision and a monthly test.' },
							{
								title: 'Dakhil and Alim',
								text: 'The full board syllabus, taught to the board exam.'
							},
							{ title: 'Evening classes', text: 'For working students and for the neighbourhood.' }
						]
					},
					{ type: 'courses', heading: 'Online lessons', limit: 6 },
					{
						type: 'stats',
						items: [
							{ title: '340', text: 'students' },
							{ title: '62', text: 'huffaz since 2010' },
							{ title: '22', text: 'teachers' }
						]
					},
					{
						type: 'notices',
						heading: 'Notices',
						items: [
							{ title: 'Ramadan timetable published', text: '2 Ramadan' },
							{ title: 'Hifz sanad ceremony', text: '15 Shawwal' }
						]
					},
					{
						type: 'cta',
						heading: 'Admissions for the new session are open',
						text: 'Places for the hifz section are limited to twenty a year.',
						cta_label: 'Create an account',
						cta_href: '/login'
					}
				]
			},
			{
				slug: 'about',
				title: 'About the madrasah',
				nav_label: 'About',
				nav_order: 1,
				description: 'Who teaches here and what the day looks like.',
				blocks: [
					{
						type: 'richtext',
						heading: 'A word from the principal',
						text: 'We began in one room with eleven students. What has not changed is the rule that a child is taught by name, not by number.\n\nThe day begins with fajr and ends after maghrib, with rest, meals and games in between.'
					},
					{
						type: 'features',
						heading: 'The teachers',
						items: [
							{ title: 'Hafiz Abdur Rahman', text: 'Hifz section, twenty-one years here.' },
							{ title: 'Maulana Ibrahim', text: 'Fiqh and Arabic, Alim section.' },
							{ title: 'Mrs Salma Khatun', text: 'Mathematics and science.' }
						]
					}
				]
			}
		]
	},
	{
		id: 'bengal-college',
		name: 'Bengal college',
		kind: 'college',
		region: 'bengal',
		theme: 'bengal',
		summary: 'HSC groups, results and the admission notice.',
		pages: [
			{
				slug: '',
				title: 'Your college',
				nav_label: '',
				nav_order: 0,
				description: 'An HSC college with science, business studies and humanities.',
				blocks: [
					{
						type: 'hero',
						heading: 'Two years that decide the next ten',
						text: 'Science, business studies and humanities, taught by people who have sat the board exam a hundred times over.',
						cta_label: 'Admission notice',
						cta_href: '/admissions'
					},
					{
						type: 'stats',
						items: [
							{ title: '2,100', text: 'students' },
							{ title: '89', text: 'teachers' },
							{ title: '18', text: 'GPA 5 last year' },
							{ title: '1986', text: 'founded' }
						]
					},
					{
						type: 'features',
						heading: 'The three groups',
						items: [
							{ title: 'Science', text: 'Physics, chemistry, biology and higher mathematics.' },
							{ title: 'Business studies', text: 'Accounting, management and finance.' },
							{ title: 'Humanities', text: 'Civics, economics, logic and Islamic history.' }
						]
					},
					{ type: 'courses', heading: 'Courses open now', limit: 6 },
					{
						type: 'notices',
						heading: 'Notices',
						items: [
							{ title: 'HSC form fill-up begins', text: '20 July' },
							{ title: 'Second-year test exam routine', text: '5 July' }
						]
					}
				]
			},
			{
				slug: 'admissions',
				title: 'Admission notice',
				nav_label: 'Admissions',
				nav_order: 1,
				description: 'Who may apply, what the merit list means, and the dates.',
				blocks: [
					{
						type: 'richtext',
						heading: 'Admission for the coming session',
						text: 'Applications open the week the SSC results are published and close a fortnight later.\n\nSelection is on the SSC GPA. The merit list is put up on this page and on the college gate.'
					},
					{
						type: 'faq',
						heading: 'Questions',
						items: [
							{
								title: 'What GPA do I need?',
								text: 'Science asks 4.00, business studies 3.50, humanities 3.00.'
							},
							{
								title: 'Is there a hostel?',
								text: 'For girls, on the campus. For boys, four approved messes nearby.'
							}
						]
					}
				]
			}
		]
	},
	{
		id: 'bengal-university',
		name: 'Bengal university',
		kind: 'university',
		region: 'bengal',
		theme: 'bengal',
		summary: 'Faculties, research and the admission test.',
		pages: [
			{
				slug: '',
				title: 'Your university',
				nav_label: '',
				nav_order: 0,
				description: 'A university with faculties in science, business and the arts.',
				blocks: [
					{
						type: 'hero',
						heading: 'Where the question matters more than the answer',
						text: 'Undergraduate and master’s degrees across seven faculties, taught in Dhaka.',
						cta_label: 'Programmes',
						cta_href: '/programmes'
					},
					{
						type: 'stats',
						items: [
							{ title: '11,400', text: 'students' },
							{ title: '640', text: 'faculty' },
							{ title: '7', text: 'faculties' },
							{ title: '1971', text: 'founded' }
						]
					},
					{ type: 'courses', heading: 'Open for enrollment', limit: 6 },
					{
						type: 'features',
						heading: 'Research that leaves the campus',
						items: [
							{
								title: 'Water and climate',
								text: 'Salinity in the coastal belt, with three district councils.'
							},
							{ title: 'Public health', text: 'Field studies with the upazila health complexes.' },
							{ title: 'Language technology', text: 'Bangla speech and text, released openly.' }
						]
					},
					{
						type: 'cta',
						heading: 'The admission test is in December',
						text: 'Registration opens in October. Past papers are on the programmes page.',
						cta_label: 'Create an account',
						cta_href: '/login'
					}
				]
			},
			{
				slug: 'programmes',
				title: 'Programmes',
				nav_label: 'Programmes',
				nav_order: 1,
				description: 'What you can read here, and for how long.',
				blocks: [
					{
						type: 'features',
						heading: 'Undergraduate',
						items: [
							{ title: 'BSc in computer science', text: 'Four years, eight semesters.' },
							{ title: 'BBA', text: 'Four years, with an internship in the last.' },
							{ title: 'BA in English', text: 'Four years, literature and linguistics.' }
						]
					},
					{
						type: 'features',
						heading: 'Postgraduate',
						items: [
							{ title: 'MSc in data science', text: 'Eighteen months, evening classes available.' },
							{ title: 'MBA', text: 'Two years, or one for a BBA graduate.' }
						]
					}
				]
			}
		]
	},
	{
		id: 'gulf-school',
		name: 'Gulf school',
		kind: 'school',
		region: 'gulf',
		theme: 'gulf',
		summary: 'Arabic-first, with the national curriculum and admissions.',
		pages: [
			{
				slug: '',
				title: 'مدرستكم',
				nav_label: '',
				nav_order: 0,
				description: 'مدرسة تعلّم المنهج الوطني مع القرآن الكريم.',
				blocks: [
					{
						type: 'hero',
						heading: 'تعليم يبني الإنسان قبل الدرجة',
						text: 'المنهج الوطني كاملًا، مع حلقة قرآن يومية لكل صف.',
						cta_label: 'التسجيل',
						cta_href: '/admissions'
					},
					{
						type: 'stats',
						items: [
							{ title: '860', text: 'طالبًا وطالبة' },
							{ title: '54', text: 'معلمًا' },
							{ title: '١٩٩٨', text: 'سنة التأسيس' }
						]
					},
					{
						type: 'features',
						heading: 'لماذا نحن',
						items: [
							{ title: 'صفوف صغيرة', text: 'خمسة وعشرون طالبًا في الصف الواحد.' },
							{ title: 'حلقة قرآن', text: 'حفظ ومراجعة كل يوم قبل الحصص.' },
							{ title: 'تواصل مع الأسرة', text: 'يصلكم الغياب في نفس الصباح.' }
						]
					},
					{ type: 'courses', heading: 'الدروس على المنصّة', limit: 6 },
					{
						type: 'cta',
						heading: 'التسجيل مفتوح للفصل القادم',
						text: 'المقاعد محدودة في الصفوف الأولى.',
						cta_label: 'إنشاء حساب',
						cta_href: '/login'
					}
				]
			},
			{
				slug: 'admissions',
				title: 'التسجيل',
				nav_label: 'التسجيل',
				nav_order: 1,
				description: 'شروط القبول والرسوم والمواعيد.',
				blocks: [
					{
						type: 'richtext',
						heading: 'كيف تسجّلون أبناءكم',
						text: 'يبدأ التسجيل في شهر مايو ويستمر حتى امتلاء المقاعد.\n\nالمطلوب: شهادة الميلاد، صورة الهوية، وآخر تقرير دراسي.'
					},
					{
						type: 'faq',
						heading: 'أسئلة متكررة',
						items: [
							{ title: 'ما قيمة الرسوم؟', text: 'تُدفع فصليًا، ويمكن تقسيطها على ثلاث دفعات.' },
							{ title: 'هل يوجد نقل مدرسي؟', text: 'نعم، يغطي أحياء المدينة الرئيسية.' }
						]
					}
				]
			}
		]
	},
	{
		id: 'gulf-madrasah',
		name: 'Gulf institute of Qur’an',
		kind: 'madrasah',
		region: 'gulf',
		theme: 'gulf',
		summary: 'Hifz circles, ijazah and tajweed, in Arabic.',
		pages: [
			{
				slug: '',
				title: 'معهدكم',
				nav_label: '',
				nav_order: 0,
				description: 'معهد لتحفيظ القرآن الكريم وإجازات الرواية.',
				blocks: [
					{
						type: 'hero',
						heading: 'حلقات قرآنية بسند متصل',
						text: 'حفظ وتجويد وإجازة، للرجال والنساء، حضورًا وعن بُعد.',
						cta_label: 'الحلقات',
						cta_href: '/courses'
					},
					{
						type: 'features',
						heading: 'ما نقدّمه',
						items: [
							{ title: 'حلقات الحفظ', text: 'خمسة أيام في الأسبوع، مع مراجعة أسبوعية.' },
							{ title: 'التجويد', text: 'من المبتدئ إلى المتقن، مع تسميع فردي.' },
							{ title: 'الإجازة', text: 'سند متصل إلى النبي صلى الله عليه وسلم.' }
						]
					},
					{
						type: 'stats',
						items: [
							{ title: '410', text: 'طالبًا' },
							{ title: '73', text: 'إجازة ممنوحة' },
							{ title: '19', text: 'معلمًا ومعلمة' }
						]
					},
					{ type: 'courses', heading: 'التسجيل المفتوح', limit: 6 },
					{
						type: 'cta',
						heading: 'ابدأ حلقتك هذا الأسبوع',
						text: 'اختبار تحديد المستوى يستغرق عشر دقائق.',
						cta_label: 'إنشاء حساب',
						cta_href: '/login'
					}
				]
			},
			{
				slug: 'about',
				title: 'عن المعهد',
				nav_label: 'عن المعهد',
				nav_order: 1,
				description: 'نشأة المعهد ومنهجه في التعليم.',
				blocks: [
					{
						type: 'richtext',
						heading: 'كلمة المشرف',
						text: 'بدأنا بحلقة واحدة في المسجد، واليوم نعلّم مئات الطلاب.\n\nمنهجنا بسيط: قليل متقن خير من كثير متروك.'
					}
				]
			}
		]
	},
	{
		id: 'gulf-college',
		name: 'Gulf college',
		kind: 'college',
		region: 'gulf',
		theme: 'gulf',
		summary: 'Diplomas and professional training, Arabic-first.',
		pages: [
			{
				slug: '',
				title: 'كليتكم',
				nav_label: '',
				nav_order: 0,
				description: 'كلية تقدّم دبلومات مهنية معتمدة.',
				blocks: [
					{
						type: 'hero',
						heading: 'دبلومات تُوظِّف، لا شهادات تُعلَّق',
						text: 'برامج مهنية قصيرة في المحاسبة والتقنية والإدارة، بشراكة مع سوق العمل.',
						cta_label: 'البرامج',
						cta_href: '/programmes'
					},
					{
						type: 'stats',
						items: [
							{ title: '1,900', text: 'طالبًا' },
							{ title: '87%', text: 'نسبة التوظيف' },
							{ title: '24', text: 'برنامجًا' }
						]
					},
					{ type: 'courses', heading: 'البرامج المتاحة', limit: 6 },
					{
						type: 'faq',
						heading: 'أسئلة',
						items: [
							{ title: 'كم مدة الدبلوم؟', text: 'من فصل واحد إلى أربعة فصول حسب البرنامج.' },
							{ title: 'هل الدراسة مسائية؟', text: 'أغلب البرامج لها شعبة مسائية.' }
						]
					}
				]
			},
			{
				slug: 'programmes',
				title: 'البرامج',
				nav_label: 'البرامج',
				nav_order: 1,
				description: 'قائمة الدبلومات ومدتها.',
				blocks: [
					{
						type: 'features',
						heading: 'الدبلومات',
						items: [
							{ title: 'المحاسبة', text: 'أربعة فصول، مع تدريب ميداني.' },
							{ title: 'الشبكات', text: 'ثلاثة فصول، مع شهادات مهنية.' },
							{ title: 'إدارة المكاتب', text: 'فصلان.' }
						]
					}
				]
			}
		]
	},
	{
		id: 'gulf-university',
		name: 'Gulf university',
		kind: 'university',
		region: 'gulf',
		theme: 'gulf',
		summary: 'Colleges, research and graduate admission, in Arabic.',
		pages: [
			{
				slug: '',
				title: 'جامعتكم',
				nav_label: '',
				nav_order: 0,
				description: 'جامعة تضم كليات في العلوم والشريعة والهندسة.',
				blocks: [
					{
						type: 'hero',
						heading: 'جامعة تبحث وتُعلّم',
						text: 'كليات في الشريعة والعلوم والهندسة، ودراسات عليا في اثني عشر تخصصًا.',
						cta_label: 'الكليات',
						cta_href: '/colleges'
					},
					{
						type: 'stats',
						items: [
							{ title: '14,800', text: 'طالبًا' },
							{ title: '910', text: 'عضو هيئة تدريس' },
							{ title: '12', text: 'برنامج دراسات عليا' },
							{ title: '١٩٨٤', text: 'سنة التأسيس' }
						]
					},
					{ type: 'courses', heading: 'مقررات مفتوحة', limit: 6 },
					{
						type: 'features',
						heading: 'مراكز البحث',
						items: [
							{ title: 'مركز الدراسات القرآنية', text: 'المخطوطات والقراءات.' },
							{ title: 'مركز المياه', text: 'التحلية وإدارة الموارد.' },
							{ title: 'مركز الذكاء الاصطناعي', text: 'معالجة اللغة العربية.' }
						]
					},
					{
						type: 'cta',
						heading: 'القبول للدراسات العليا مفتوح',
						text: 'يغلق باب التقديم في نهاية الفصل الأول.',
						cta_label: 'إنشاء حساب',
						cta_href: '/login'
					}
				]
			},
			{
				slug: 'colleges',
				title: 'الكليات',
				nav_label: 'الكليات',
				nav_order: 1,
				description: 'كليات الجامعة وأقسامها.',
				blocks: [
					{
						type: 'features',
						heading: 'الكليات',
						items: [
							{ title: 'كلية الشريعة', text: 'الفقه وأصوله، والدراسات القرآنية.' },
							{ title: 'كلية العلوم', text: 'الرياضيات والفيزياء والأحياء.' },
							{ title: 'كلية الهندسة', text: 'المدنية والكهربائية والحاسب.' }
						]
					}
				]
			}
		]
	}
];

export const regions = [
	{ value: 'bengal', label: 'Bangladesh and South Asia' },
	{ value: 'gulf', label: 'The Gulf' }
] as const;

export const kindNames: Record<SiteTemplate['kind'], string> = {
	school: 'School',
	college: 'College',
	madrasah: 'Madrasah',
	university: 'University'
};
