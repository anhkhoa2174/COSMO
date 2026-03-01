'use client';

import Link from 'next/link';

const Product = () => (
  <div className="py-8">
    <div id="product" className="container mx-auto">
      <div className="mt-4 text-center">
        <h1 className="text-4xl font-bold">
          Convert leads into revenue with an AI workforce
        </h1>
        <div className="mt-4">
          <Link
            href="https://e3tbtbnfxh7.typeform.com/to/TxJsQ5wz"
            target="_blank"
          >
            <button
              className="rounded-md bg-gradient-to-r from-blue-500 to-purple-500 px-6 py-3 font-medium text-white shadow-lg"
              style={{
                boxShadow: '0px 16px 20px rgba(98, 87, 243, 0.35)',
              }}
            >
              Join our waitlist
            </button>
          </Link>
        </div>
      </div>
      <div className="w-full">
        <img
          src="/landing-page/header.png"
          alt="header.png"
          className="w-full"
        />
      </div>
      <div className="mt-4">
        <div className="flex flex-wrap items-center justify-between">
          <div>
            <p className="text-lg font-semibold text-gray-500">
              Built by a team
            </p>
            <p className="text-lg font-semibold">from</p>
          </div>
          <img
            src="https://logos-download.com/wp-content/uploads/2016/02/Google_Logo_2015-450x148.png"
            alt="Google"
            className="h-10"
          />
          <img
            src="https://www.obilityb2b.com/wp-content/uploads/2017/07/Intuit_logo_logotype.png"
            alt="Intuit"
            className="h-10"
          />
          <img
            src="https://logos-download.com/wp-content/uploads/2016/10/Zendesk_logo_wordmark-700x196.png"
            alt="Zendesk"
            className="h-10"
          />
          <img
            src="https://irp-cdn.multiscreensite.com/762ccab2/dms3rep/multi/Indeed-Logo.png"
            alt="Indeed"
            className="h-10"
          />
        </div>
      </div>
    </div>
  </div>
);

export default Product;
