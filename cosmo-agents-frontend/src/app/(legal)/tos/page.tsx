'use client';

import { useEffect, useRef, useState } from 'react';

import Markdown from 'react-markdown';
import rehypeRaw from 'rehype-raw';
import remarkGfm from 'remark-gfm';

const getId = (text: string) =>
  text
    .toLowerCase()
    .replace(/[^\w]+/g, '-')
    .replace(/(^-|-$)/g, '');

export default function TermsOfService() {
  const mdRef = useRef<HTMLDivElement>(null);
  const [headings, setHeadings] = useState<
    { id: string; text: string; level: number }[]
  >([]);

  useEffect(() => {
    if (!mdRef.current) return;
    const headingElements = mdRef.current.querySelectorAll('h2');

    const data = Array.from(headingElements).map((el) => {
      const text = el.textContent || '';
      const id = getId(text);
      el.id = id; // Set the id for the heading element
      return {
        text,
        id,
        level: parseInt(el.tagName[1]), // 'H1' => 1, 'H2' => 2, etc.
      };
    });

    setHeadings(data);
  }, [term]);

  return (
    <div>
      <div className="flex h-full">
        <aside className="fixed left-4 top-28 w-[300px] rounded-2xl border-2 border-accent-foreground p-2">
          <ul className="space-y-1 text-sm">
            {headings.map((h) => (
              <li
                key={h.id}
                className={`ml-${(h.level - 1) * 4} cursor-pointer rounded-md px-4 py-2 hover:bg-zinc-100`}
                onClick={(e) => {
                  e.preventDefault();
                  const el = document.getElementById(h.id);
                  if (el) {
                    el.scrollIntoView({ behavior: 'smooth', block: 'start' });
                  }
                }}
              >
                {h.text}
              </li>
            ))}
          </ul>
        </aside>
        <div className="ml-[332px] p-4">
          <div
            className="container mx-auto text-base leading-loose"
            ref={mdRef}
          >
            <Markdown
              remarkPlugins={[remarkGfm]}
              rehypePlugins={[rehypeRaw]}
              components={{
                p: ({ node, ...props }) => <p className="mb-2" {...props} />,
                h1: ({ node, ...props }) => (
                  <h1 className="my-5 text-3xl font-bold" {...props} />
                ),
                h2: ({ node, children, ...props }) => {
                  const text = children?.toString() || '';
                  const id = getId(text);
                  return (
                    <h2 id={id} className="my-4 text-2xl font-bold" {...props}>
                      {children}
                    </h2>
                  );
                },
                h3: ({ node, ...props }) => (
                  <h3 className="my-3 text-xl font-bold" {...props} />
                ),
                ul: ({ node, ...props }) => (
                  <ul className="list-disc pl-5" {...props} />
                ),
                ol: ({ node, ...props }) => (
                  <ol className="list-decimal pl-5" {...props} />
                ),
                li: ({ node, ...props }) => <li className="mb-1" {...props} />,
              }}
            >
              {term}
            </Markdown>
          </div>
        </div>
      </div>
    </div>
  );
}

const term = `
# Cosmo Terms of Service

> **Prototype notice.** COSMO is a research prototype built as a university
> project. This document is a draft prepared for the intended operating entity
> and is **not a binding agreement**: no commercial service is offered under it
> and no entity is named as a party until the product is released. It is
> published so that anyone testing COSMO can see exactly how the system handles
> data.

Thanks for your interest in Cosmo.

In these Terms, 'Company', 'we', 'us', or 'our' refers to the team operating COSMO. The operating entity, its registration details and its place of establishment will be stated here before any commercial release.

These Terms of Service, together with any amendments, order forms, and any additional agreements you enter into with Cosmo in connection with the Service (collectively, "Terms" or "Agreement"), govern your access to and use of Cosmo ("Cosmo", "we" or "our") websites, services, and applications (collectively, the "Services"). This Agreement is effective between You and the COSMO team as of the earlier of the date both You and Cosmo executed the Order Form referencing these Terms of Service or the date You clicked your acceptance ("Effective Date") and may be amended only as set forth herein.

These Terms apply to all Cosmo customers and users of the Service. Please read them carefully before using the Service.

By accessing or using the Service you agree to be bound by these Terms. If you are using the Service on behalf of an organization or entity ("Organization"), then you are agreeing to these Terms on behalf of that Organization, and you represent and warrant that you have the authority to bind the Organization to these Terms. In that case, "you" and "your" refers to you and that Organization.

In order to use our Services you must link a 3rd party email account to your Cosmo Services account. With your permission (which you are granting by using the Services), we will, within the Services, create, modify, and update email content that will be delivered to you through your email systems. You will also have the option to import lists of your leads through the Services. This data, along with any electronic data and information submitted by or for you to the Services, including electronic data and information submitted by or for you through your use of third party applications, or collected and processed by or for you using the Services (excluding information obtained by the Company from our content licensors or publicly available sources and provided to you, or otherwise provided by the Company to you in connection with the Cosmo Services) is referred to as "Your Data." You retain full ownership of Your Data. You are responsible for obtaining all approvals necessary for Cosmo to use Your Data to provide the Services as authorized under these Terms. Cosmo may use Your Data to provide the Services to you. In addition, we may (i) process usage and performance data with respect to the use and performance of the Services, and (ii) use and analyze Your Data on a de-identified/aggregated basis for our internal business purposes, including improving, testing and providing our services. Cosmo may only disclose usage and performance related data in a de-identified and aggregated form (e.g., not specifically identifying Customer, or any customer or prospect of Customer), for example, to describe best practices or publish general information and statistics regarding the services.

Cosmo will respond to requests to transfer or delete Your Data only to the extent such requests are addressed to Cosmo from an email address from the email domain that is the same as the domain associated with your Cosmo account or from the in-app support chat with an authenticated account. To the extent the request does not originate from the specific email address associated with your Cosmo account (the "Authorized Email"), Cosmo will notify you of the request at the Authorized Email, and Cosmo will deem such request valid unless it receives a response within 5 business days from the Authorized Email objecting to the request. Cosmo will deem valid any request addressed to Cosmo from the Authorized Email, and you are solely responsible for the validity of all requests or communications addressed to Cosmo from the Authorized Email.

We will maintain administrative, physical, and technical safeguards designed to protect the security, confidentiality and integrity of Your Data as described in the Privacy section of these Terms and in the Cosmo Data Processing Addendum. You are solely responsible for protecting your passwords, limiting access to your computers and devices, and signing out of the Cosmo Services when you are not using them.

## Pricing & Payment

The fees for the Service shall be paid in accordance with the terms set forth on the applicable Services Order Form.

## Privacy

### Current Data Usage

We explicitly affirm that we DO NOT use any data obtained through Google Workspace APIs (including Gmail) to develop, improve, or train generalized or non-personalized artificial intelligence or machine learning models.

### Third-Party AI Processing

We use the following AI providers to process your data:

- **OpenAI** (U.S.) — language models and text embeddings for classification, drafting, enrichment and semantic search
- **Anthropic** (U.S.) — language models for the in-app business-development assistant
- **Coze / ByteDance** (U.S.) — the agent platform behind the in-app AI writer and reply drafting

Specific data transferred to these providers includes:

- Email content: subject lines and bodies of inbound replies and the conversation thread being answered, including sender names and email addresses as they appear in that thread
- Contact records: name, job title, company, industry, email address, LinkedIn URL and enrichment attributes, where relevant to the request
- Interaction history and internal notes your team records against a contact
- Company knowledge you upload to Cosmo, and documents or images you upload for analysis
- Web page text retrieved from public URLs you ask Cosmo to look at

Email attachments received in your mailbox are never sent to AI providers. Files you deliberately upload to Cosmo are.

Purpose of transfer:

- Classification of inbound replies into a fixed set of intents
- Generating grounded draft email replies and outreach content
- Contact enrichment, semantic search and meeting preparation
- Powering the in-app AI writing assistant

Important data handling notes:

- All transfers are encrypted in transit
- Reply-drafting calls are made without conversation persistence on the provider side; in-app AI writer sessions create a conversation with the provider to maintain chat context
- No data is used for training AI models by us, and our provider agreements restrict training on your data
- Content received from prospects and retrieved documents are treated as untrusted data: in the reply pipeline they are isolated in delimited context blocks, classifier output is validated against a fixed intent set, and an instruction injected inside an email cannot cause a message to be sent
- AI-drafted replies to inbound mail are held for review and sent by a person. Campaign and playbook sequences that you build and activate send on the schedule you configure
- You can stop AI processing at any time by disconnecting your linked mailbox or contacting support

### Email Intent Classification

Cosmo includes a production email intent classification feature: inbound replies on active campaign threads in a connected mailbox are classified into one of nine intents (interested, not interested, request for pricing, request for information, referral, nurture, out of office, do not contact, and unknown — which is routed to a person) and dispatched to a matching action. Classification:

- Runs as each reply is processed; no separate training pipeline consumes your email
- Uses the email body of the inbound reply; attachments are not used
- Honors opt-out: a do-not-contact reply flags the contact, and every automated send path skips them from that point on
- Data deletion requests are actioned by support; see the Data Processing Addendum for timelines

### Custom Client Models

- Only developed with explicit client request and consent
- Models will be client-specific, using only their data
- Will require Google approval before implementation
- Separate agreements will govern custom model development

### Commitment to Privacy

We protect your Google user data by:

- No collection for AI/ML training
- No building of generalized AI models
- Inference only — we send data to providers to answer a request, not to build datasets. Derived summaries and search embeddings are stored inside Cosmo so retrieval works; automated expiry of those derived records is on our roadmap
- Data-processing agreements in place with providers

### Updates

We will notify users and seek Google approval for any changes to these practices.

We care about the privacy of our users. We may collect, use and share Your Data that includes personally identifiable information as described in the Privacy section of these Terms and the Cosmo Data Processing Addendum ("Cosmo DPA"). Except as may be otherwise identified in those documents, we won't share such personal information with others unless: (a) you have given us permission to do so; (b) we are required to by law or by valid legal process; (c) we need to do so in order to provide you the Service; or (d) one of the other exceptions described in these Terms or the Cosmo DPA applies. To the extent that in connection with the performance of the Services, Cosmo processes on your behalf any Personal Data subject to the Applicable Laws (as defined in the Cosmo DPA) contained in Your Data, the terms of the Cosmo DPA (or such other DPA executed by both parties that references this Agreement), shall apply and the parties agree to comply with such terms. The DPA may be updated by Cosmo if required by applicable law.

**Q: What AI technology does Cosmo use?**

We use current OpenAI language and embedding models for reply intent classification, retrieval-grounded drafting and enrichment, and the Coze agent platform for the in-app AI writing assistant.

**Q: How does Cosmo handle data with AI currently?**

- We process data solely to provide the outreach features you signed up for
- Email content is processed as-is for your automated email communications; prospect-written content is isolated as untrusted data and cannot instruct the AI to send a message
- AI-drafted replies to inbound mail require human approval before sending; campaign and playbook sequences send on the schedule you configure once you activate them

**Q: What about data usage for model training?**

- No customer data is used to train AI models
- If that ever changes, customers will be notified in advance, given control options, and any such use will follow strict privacy guidelines

**Q: What safeguards protect my data during AI processing?**

- Access scoped per user and organization, with short-lived access tokens
- Untrusted-data isolation and a fixed output space for classification
- Review of AI-drafted replies before they are sent
- Provider agreements that prohibit training on your data
- Single sign-on, multi-factor authentication and application-level encryption of stored credentials are on our roadmap

## Access & Data Security

You give us permission to access your computer, or other telecommunications or information systems ("Systems") in order to provide the Service. This permission is limited to those Systems, time periods, and personnel as are reasonably needed to provide the Service. Access is subject to business control and information protection policies, standards, and guidelines designed to ensure that access granted hereunder will not impair the integrity and availability of your Systems.

We shall implement and maintain reasonable administrative, physical and technical safeguards that are designed to prevent any unauthorized use, access, processing, destruction, loss, alteration, or disclosure of Your Data (including any applicant or employee data furnished by you as may be held or accessed by us). And we shall notify you as soon as reasonably possible following discovery of any breach or compromise of the security, confidentiality, or integrity of Your Data.

## Responsible Disclosure Policy

Cosmo aims to keep its Services safe for everyone, and we consider data security to be of the utmost importance. If you are a security researcher and have discovered a security vulnerability in the Services, we appreciate your help in disclosing it to us in a responsible manner at security@cosmoagents.ai.

## Limitation of Liability

To the fullest extent allowed by applicable law, you agree to indemnify and hold Cosmo, its affiliates, officers, agents, employees, suppliers, licensors and partners harmless from and against any and all claims, liabilities, damages (actual and consequential), losses and expenses (including attorneys' fees) arising from any third party claims relating to (a) your use of the Service (including any actions taken by a third party using your account), or (b) your violation of these Terms. In the event of such a claim, suit, or action ("Claim"), we will attempt to provide notice of the Claim to the contact information we have for your account (provided that failure to deliver such notice shall not eliminate or reduce your indemnification obligations hereunder).

LIMITATION OF LIABILITY. EXCEPT IN THE EVENT OF GROSS NEGLIGENCE OR WILLFUL MISCONDUCT, IN NO EVENT WILL EITHER PARTY'S AND ITS SUPPLIERS BE LIABLE TO THE OTHER PARTY, ITS AFFILIATES, USERS OR ANY OTHER THIRD PARTY FOR ANY LOSS OF PROFITS, LOSS OF USE, LOSS OF REVENUE, LOSS OF GOODWILL, LOSS OF CUSTOMER DATA OR CUSTOMER'S SOFTWARE (OR ANY DATA RELATED THERETO) OR ANY INTERRUPTION OF BUSINESS, OR FOR ANY INDIRECT, SPECIAL, INCIDENTAL, EXEMPLARY, PUNITIVE OR CONSEQUENTIAL DAMAGES OF ANY KIND ARISING OUT OF OR IN CONNECTION WITH THIS AGREEMENT OR THE SERVICES, REGARDLESS OF THE FORM OF ACTION, WHETHER IN CONTRACT, TORT, STRICT LIABILITY OR OTHERWISE, EVEN IF A PARTY HAS BEEN ADVISED OR IS OTHERWISE AWARE OF THE POSSIBILITY OF SUCH DAMAGES. THE FOREGOING DISCLAIMER WILL APPLY TO THE MAXIMUM EXTENT PERMITTED BY APPLICABLE LAW. IN NO EVENT WILL COSMO AND ITS SUPPLIERS' TOTAL AGGREGATE LIABILITY ARISING OUT OF OR RELATED TO THIS AGREEMENT EXCEED THE SUBSCRIPTION FEES PAID BY YOU TO COSMO DURING THE TWELVE (12) MONTHS PRECEDING THE INITIAL CLAIM. MULTIPLE CLAIMS WILL NOT EXPAND THIS LIMITATION. THE FOREGOING DISCLAIMER WILL APPLY TO THE MAXIMUM EXTENT PERMITTED BY APPLICABLE LAW.

## Miscellaneous

Governing law and jurisdiction will be those of the place where the operating entity is established, to be stated here before any commercial release. Until then, no dispute-resolution forum is agreed, because no commercial service is offered under this document.

Neither this Agreement nor any right or duty under this Agreement may be transferred, assigned or delegated by You, including in connection with a corporate reorganization, merger, acquisition or other change in control, without the prior written consent of Cosmo. Cosmo may assign this Agreement, including to its affiliates or in connection with a corporate reorganization, merger, acquisition or other change in control. Subject to the foregoing, this Agreement will be binding upon and will inure to the benefit of the parties and their respective representatives, heirs, administrators, successors and permitted assigns.

These Terms, together with each Order Form, is the entire agreement of the parties regarding the subject matter hereof, superseding all other agreements between them, whether oral or written. Cosmo may update or revise these Terms from time to time in its sole discretion without notice to you, and Cosmo recommends that you review these Terms on a regular basis to stay abreast of the most current version. The most current version will be posted on the site. Your continued use of the Services after any update or revision to these Terms constitutes your acceptance of the updates or revisions. Except as expressly stated in these Terms, no terms or conditions stated in a Customer purchase order or other Customer ordering document (other than with respect to duration, service and pricing that are consistent with the applicable executed Order Form) shall be incorporated into or form any part of these Terms (notwithstanding any language to the contrary therein), and all such terms or conditions shall be null and void. Order Forms governed by these Terms may be executed in one or more counterparts, each of which when so executed and delivered or transmitted by facsimile, e-mail or other electronic means, shall be deemed to be an original and all of which taken together shall constitute but one and the same instrument. A facsimile or electronic signature is deemed an original signature for all purposes under these Terms. All headings contained in this Agreement are inserted for identification and convenience and will not be deemed part of this Agreement for purposes of interpretation. All remedies set forth in this Agreement are cumulative.
`;
