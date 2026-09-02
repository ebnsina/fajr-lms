declare global {
	namespace App {
		interface Locals {
			token?: string;
			tenantSlug?: string;
			dir?: 'ltr' | 'rtl';
			lang?: string;
		}
	}
}

export {};
