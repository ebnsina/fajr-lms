package demo

import "github.com/ebnsina/fajr-lms/internal/database"

// The five schools the demo offers, one for each kind of place that asks. The
// content is what makes them different: the spine underneath is the same.

var madrasah = Kind{
	Slug: "madrasah", Name: "Darul Huda Madrasah", Label: "A madrasah",
	Tenant: database.TenantKindInstitution, Institution: database.InstitutionKindMadrasah,
	Dir: database.TextDirAuto, Locale: "bn", Currency: "BDT", Attends: true,
	Learners: []string{
		"Mufti Abdur Rahman", "Fatima Rahman", "Yusuf Islam", "Khadija Akter",
		"Bilal Hossain", "Maryam Sultana", "Ibrahim Chowdhury",
	},
	Courses: []Course{
		{
			Title: "Hifz, Juz Amma", Summary: "Memorisation, revision and the daily sabaq, tracked page by page.",
			Modules: []Module{
				{Title: "Sabaq", Lessons: []Lesson{
					{Title: "An-Nas to Al-Falaq", Kind: database.LessonKindAudio, Body: "Listen, then recite back to your teacher."},
					{Title: "Al-Ikhlas to Al-Masad", Kind: database.LessonKindAudio, Body: "Three repetitions, then the whole page."},
					{Title: "An-Nasr to Al-Kafirun", Kind: database.LessonKindAudio, Body: "Watch the madd; do not rush the endings."},
				}},
				{Title: "Sabqi and manzil", Lessons: []Lesson{
					{Title: "Yesterday's page, from memory", Kind: database.LessonKindText, Body: "Recite without looking. Mark what slipped."},
					{Title: "The week's revision", Kind: database.LessonKindQuiz, Body: "Ten lines, chosen at random."},
				}},
			},
			Quiz: Quiz{Title: "Tajwid, the rules of nun sakinah", Questions: []Question{
				{Prompt: "Nun sakinah before a ba is read as", Options: []string{"Iqlab", "Idgham", "Izhar", "Ikhfa"}, Correct: 0},
				{Prompt: "Which letter group calls for izhar?", Options: []string{"The throat letters", "Ya, ra, mim, lam, waw, nun", "Ba alone", "The remaining fifteen"}, Correct: 0},
				{Prompt: "A madd of two counts is", Options: []string{"Madd asli", "Madd lazim", "Madd muttasil", "Madd munfasil"}, Correct: 0},
			}},
		},
		{
			Title: "Arabic grammar, first year", Summary: "Nahw from the ground up: the noun, the verb, and the sentence they make.",
			Modules: []Module{
				{Title: "The word", Lessons: []Lesson{
					{Title: "Ism, fi'l, harf", Kind: database.LessonKindVideo, Body: "Every word in Arabic is one of three."},
					{Title: "Signs of the noun", Kind: database.LessonKindText, Body: "Tanwin, the definite article, and what follows a preposition."},
				}},
				{Title: "The sentence", Lessons: []Lesson{
					{Title: "Mubtada and khabar", Kind: database.LessonKindVideo, Body: "The nominal sentence, with worked examples."},
					{Title: "Practice, twenty sentences", Kind: database.LessonKindAssignment, Body: "Parse each one and hand it in."},
				}},
			},
		},
		{
			Title: "Fiqh of salah", Summary: "Purification, the conditions of prayer, and what breaks it.",
			Modules: []Module{{Title: "Taharah", Lessons: []Lesson{
				{Title: "Wudu, step by step", Kind: database.LessonKindVideo, Body: "The obligatory acts and the sunnahs."},
				{Title: "When wudu is broken", Kind: database.LessonKindText, Body: "The agreed cases, and the disputed ones."},
			}}},
		},
	},
}

var school = Kind{
	Slug: "school", Name: "Shaheen Model School", Label: "A school or college",
	Tenant: database.TenantKindInstitution, Institution: database.InstitutionKindSchool,
	Dir: database.TextDirAuto, Locale: "bn", Currency: "BDT", Attends: true,
	Learners: []string{
		"Nasrin Jahan", "Tanvir Ahmed", "Sadia Islam", "Rafiq Uddin",
		"Mim Akter", "Sabbir Hasan", "Nusrat Jahan", "Arif Mahmud",
	},
	Courses: []Course{
		{
			Title: "Class 9 Mathematics", Summary: "Algebra, geometry and trigonometry, to the board syllabus.",
			Modules: []Module{
				{Title: "Algebra", Lessons: []Lesson{
					{Title: "Factorising quadratics", Kind: database.LessonKindVideo, Body: "Four methods, and when each is quickest."},
					{Title: "Simultaneous equations", Kind: database.LessonKindVideo, Body: "Substitution and elimination."},
					{Title: "Problem set 1", Kind: database.LessonKindAssignment, Body: "Twelve questions. Show your working."},
				}},
				{Title: "Geometry", Lessons: []Lesson{
					{Title: "Circle theorems", Kind: database.LessonKindVideo, Body: "Six theorems and the proofs asked for in the exam."},
					{Title: "Chapter test", Kind: database.LessonKindQuiz, Body: "Twenty minutes, calculators away."},
				}},
			},
			Quiz: Quiz{Title: "Circle theorems", Questions: []Question{
				{Prompt: "The angle at the centre is", Options: []string{"Twice the angle at the circumference", "Equal to it", "Half of it", "A right angle"}, Correct: 0},
				{Prompt: "Angles in the same segment are", Options: []string{"Equal", "Supplementary", "Complementary", "Unrelated"}, Correct: 0},
				{Prompt: "A tangent meets the radius at", Options: []string{"90°", "60°", "45°", "180°"}, Correct: 0},
			}},
		},
		{
			Title: "Class 9 English", Summary: "Reading, writing and the grammar the paper actually tests.",
			Modules: []Module{{Title: "Writing", Lessons: []Lesson{
				{Title: "Paragraph and composition", Kind: database.LessonKindText, Body: "Structure first, then the sentences."},
				{Title: "Letter and email", Kind: database.LessonKindText, Body: "The formal register, with model answers."},
			}}},
		},
		{
			Title: "Class 9 Physics", Summary: "Motion, force and energy, with the practicals.",
			Modules: []Module{{Title: "Motion", Lessons: []Lesson{
				{Title: "Speed, velocity, acceleration", Kind: database.LessonKindVideo, Body: "The three graphs and how to read them."},
				{Title: "Newton's laws", Kind: database.LessonKindVideo, Body: "Each law with an example from the road."},
			}}},
		},
	},
}

var coaching = Kind{
	Slug: "coaching", Name: "Uttara Admission Coaching", Label: "A coaching centre",
	Tenant: database.TenantKindInstitution, Institution: database.InstitutionKindCoaching,
	Dir: database.TextDirAuto, Locale: "bn", Currency: "BDT", Attends: true,
	Learners: []string{
		"Shahriar Kabir", "Ayesha Siddika", "Rakib Hasan", "Tasnim Rahman",
		"Imran Khan", "Farzana Yesmin",
	},
	Courses: []Course{
		{
			Title: "Medical admission, biology", Summary: "The whole syllabus in eleven weeks, with a weekly model test.",
			PriceMinor: 450000,
			Modules: []Module{
				{Title: "Cell and its division", Lessons: []Lesson{
					{Title: "Cell structure", Kind: database.LessonKindVideo, Body: "Organelles, and the ones questions come from."},
					{Title: "Mitosis and meiosis", Kind: database.LessonKindVideo, Body: "Stage by stage, with the differences tabulated."},
					{Title: "Model test 1", Kind: database.LessonKindQuiz, Body: "Thirty questions, one hour, negative marking."},
				}},
			},
			Quiz: Quiz{Title: "Model test 1", Questions: []Question{
				{Prompt: "Crossing over happens in", Options: []string{"Prophase I", "Metaphase II", "Anaphase I", "Telophase"}, Correct: 0},
				{Prompt: "The powerhouse of the cell is the", Options: []string{"Mitochondrion", "Ribosome", "Golgi body", "Lysosome"}, Correct: 0},
				{Prompt: "Plant cell walls are made mainly of", Options: []string{"Cellulose", "Chitin", "Keratin", "Peptidoglycan"}, Correct: 0},
			}},
		},
		{
			Title: "University admission, English", Summary: "Vocabulary, grammar and the passage, drilled to time.",
			PriceMinor: 350000,
			Modules: []Module{{Title: "Grammar", Lessons: []Lesson{
				{Title: "Right form of verbs", Kind: database.LessonKindVideo, Body: "The twenty rules that cover the paper."},
				{Title: "Daily drill", Kind: database.LessonKindAssignment, Body: "Fifty questions, submitted before midnight."},
			}}},
		},
	},
}

var creator = Kind{
	Slug: "creator", Name: "Amina Teaches Quran", Label: "I teach on my own",
	Tenant: database.TenantKindCreator, Institution: database.InstitutionKindOther,
	Dir: database.TextDirAuto, Locale: "en", Currency: "BDT",
	Learners: []string{
		"Amina Begum", "Sara Ahmed", "Hamza Ali", "Zainab Karim", "Omar Farooq",
	},
	Courses: []Course{
		{
			Title: "Read the Quran in forty days", Summary: "From the letters to fluent recitation, at your own pace.",
			PriceMinor: 250000,
			Modules: []Module{
				{Title: "The letters", Lessons: []Lesson{
					{Title: "The alphabet, with sound", Kind: database.LessonKindAudio, Body: "Each letter from its point of articulation."},
					{Title: "Joining letters", Kind: database.LessonKindVideo, Body: "Beginning, middle and end forms."},
					{Title: "First reading", Kind: database.LessonKindQuiz, Body: "Recognise the letter you hear."},
				}},
				{Title: "The vowels", Lessons: []Lesson{
					{Title: "Fatha, kasra, damma", Kind: database.LessonKindAudio, Body: "Short vowels, then the long ones."},
					{Title: "Sukun and shadda", Kind: database.LessonKindVideo, Body: "Where beginners slow down, and why."},
				}},
			},
			Quiz: Quiz{Title: "First reading", Questions: []Question{
				{Prompt: "How many letters are in the Arabic alphabet?", Options: []string{"28", "26", "30", "24"}, Correct: 0},
				{Prompt: "A shadda means the letter is", Options: []string{"Doubled", "Silent", "Lengthened", "Dropped"}, Correct: 0},
				{Prompt: "Fatha is written", Options: []string{"Above the letter", "Below it", "Inside it", "After it"}, Correct: 0},
			}},
		},
		{
			Title: "Tajwid for beginners", Summary: "The rules that change how you sound in the first week.",
			PriceMinor: 150000,
			Modules: []Module{{Title: "The rules", Lessons: []Lesson{
				{Title: "Noon sakinah and tanwin", Kind: database.LessonKindVideo, Body: "Four rules, with recitation to copy."},
				{Title: "Madd, the lengthenings", Kind: database.LessonKindAudio, Body: "Counting the beats out loud."},
			}}},
		},
	},
}

var corporate = Kind{
	Slug: "corporate", Name: "Meghna Group Learning", Label: "A company",
	Tenant: database.TenantKindCorporate, Institution: database.InstitutionKindOther,
	Dir: database.TextDirAuto, Locale: "en", Currency: "BDT",
	Learners: []string{
		"Head of Learning", "Rashed Karim", "Nadia Haque", "Sohel Rana",
		"Priya Das", "Kamal Uddin",
	},
	Courses: []Course{
		{
			Title: "Workplace safety, annual refresher", Summary: "The yearly certification every floor employee must hold.",
			Modules: []Module{
				{Title: "On the floor", Lessons: []Lesson{
					{Title: "Hazards and reporting", Kind: database.LessonKindVideo, Body: "What to report, to whom, and how fast."},
					{Title: "Fire, and the way out", Kind: database.LessonKindVideo, Body: "Assembly points and the drill."},
					{Title: "Certification test", Kind: database.LessonKindQuiz, Body: "Pass at 80% to be certified for the year."},
				}},
			},
			Quiz: Quiz{Title: "Certification test", Questions: []Question{
				{Prompt: "A near miss should be reported", Options: []string{"Every time", "Only if someone was hurt", "At the month's end", "Never"}, Correct: 0},
				{Prompt: "On hearing the fire alarm you", Options: []string{"Go to the assembly point", "Finish the task", "Collect your belongings", "Use the lift"}, Correct: 0},
				{Prompt: "Personal protective equipment is", Options: []string{"Worn whenever the sign says so", "Optional in summer", "For visitors only", "For the night shift"}, Correct: 0},
			}},
		},
		{
			Title: "Anti-harassment and conduct", Summary: "The policy, what it covers, and how a complaint is handled.",
			Modules: []Module{{Title: "The policy", Lessons: []Lesson{
				{Title: "What the policy covers", Kind: database.LessonKindText, Body: "Definitions, with examples from real cases."},
				{Title: "Raising a complaint", Kind: database.LessonKindText, Body: "The committee, the timeline, and confidentiality."},
			}}},
		},
	},
}
