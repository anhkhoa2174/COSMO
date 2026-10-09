'use client';

import { useState } from 'react';

import {
  Box,
  Text,
  AppShell,
  NavLink,
  Title,
  List,
  Table,
} from '@mantine/core';

export default function DataProcessing() {
  const [active, setActive] = useState<string>('');
  return (
    <Box>
      <Box>
        <AppShell
          header={{ height: 96 }} // chừa chỗ cho navbar chung của site
          navbar={{
            width: 300,
            breakpoint: 'sm',
          }}
        >
          <AppShell.Navbar p="md">
            <NavLink
              href="#data-processing-addendum"
              label="Cosmo DATA PROCESSING ADDENDUM"
              active={active === '#data-processing-addendum'}
              fw={active === '#data-processing-addendum' ? 700 : 400}
              onClick={() => setActive('#data-processing-addendum')}
            />
            <NavLink
              href="#exhibit-a"
              label="Exhibit A"
              active={active === '#exhibit-a'}
              fw={active === '#exhibit-a' ? 700 : 400}
              onClick={() => setActive('#exhibit-a')}
            />
            <NavLink
              href="#exhibit-b"
              label="Exhibit B"
              active={active === '#exhibit-b'}
              fw={active === '#exhibit-b' ? 700 : 400}
              onClick={() => setActive('#exhibit-b')}
            />
            <NavLink
              href="#exhibit-c"
              label="Exhibit C"
              active={active === '#exhibit-c'}
              fw={active === '#exhibit-c' ? 700 : 400}
              onClick={() => setActive('#exhibit-c')}
            />
          </AppShell.Navbar>

          <AppShell.Main ta="left">
            <Box size="md" mx="128" my="lg" ta="left">
              <Title
                order={1}
                mb="md"
                id="data-processing-addendum"
                style={{ scrollMarginTop: '90px' }}
              >
                Cosmo DATA PROCESSING ADDENDUM
              </Title>
              <Text size="md" mb="md">
                Controller to Processor
              </Text>
              <Box
                mb="lg"
                p="md"
                style={{
                  border: '1px solid #FCD34D',
                  background: '#FFFBEB',
                  borderRadius: 12,
                }}
              >
                <Text size="sm" c="#78350F">
                  <strong>Prototype notice.</strong> Cosmo is a research
                  prototype built as a university project. This Addendum is a
                  draft prepared for the intended operating entity and is not a
                  binding agreement: no commercial service is offered under it,
                  and the operating entity, its registration details, address
                  and choice of law will be stated here before any commercial
                  release. It is published so that anyone testing Cosmo can see
                  how the system handles personal data, and which measures are
                  in place today versus scheduled.
                </Text>
              </Box>
              <Text size="md" mb="md">
                This Cosmo Data Processing Addendum (this “
                <Text span size="md" fw={600}>
                  Addendum
                </Text>
                ” ) is entered into by and between the team operating Cosmo
                (the operating entity is not yet incorporated; its name,
                registration details and address will be stated here before any
                commercial release) (“
                <Text span size="md" fw={600}>
                  Cosmo
                </Text>
                ”) and you (the “
                <Text span size="md" fw={600}>
                  Client
                </Text>
                ”) (each, a “
                <Text span size="md" fw={600}>
                  Party
                </Text>
                ” and, collectively, the “
                <Text span size="md" fw={600}>
                  Parties
                </Text>
                ”). This Addendum applies to every Client whose Service
                Agreement, order form, or other written agreement with Cosmo
                incorporates this Addendum by reference, and takes effect on the
                date on which that agreement takes effect (the “
                <Text span size="md" fw={600}>
                  Effective Date
                </Text>
                ”). No separate act of acceptance is required for this Addendum
                to apply. A counterpart of this Addendum for signature by both
                Parties is available on request by contacting Cosmo at
                dpo@cosmoagents.ai.
              </Text>

              <Text size="lg" mb="md" fw={600} py="lg" ta="center">
                RECITALS
              </Text>
              <Text size="md" mb="md">
                <Text span size="md" fw={600}>
                  WHEREAS
                </Text>
                , the Parties entered into the Cosmo Services Order Form, which
                incorporates by reference the Cosmo Terms of Use (collectively,
                the “
                <Text span size="md" fw={600}>
                  Service Agreement
                </Text>
                ”) and have retained the power to alter, amend, revoke, or
                terminate the Service Agreement; and
              </Text>
              <Text size="md" mb="md">
                <Text span size="md" fw={600}>
                  WHEREAS
                </Text>
                , the Parties now wish to amend the Service Agreement to ensure
                that Client Personal Data (as defined below) transferred between
                the Parties is Processed (as defined below) in compliance with
                applicable data protection principles and requirements;
              </Text>
              <Text size="md" mb="md">
                <Text span size="md" fw={600}>
                  NOW, THEREFORE
                </Text>
                , in consideration of the mutual agreements set forth in this
                Addendum, the Parties agree as follows:
              </Text>
              <List type="ordered">
                {/* 1 */}
                <List.Item c="blue">
                  <Text size="md" mb="md" c="black" fw={600}>
                    Definitions.
                  </Text>
                  <List type="ordered" listStyleType="lower-alpha">
                    {' '}
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        The definitions used in this Addendum shall have the
                        meanings set forth or referenced in this Addendum.
                        Capitalized definitions, not otherwise defined herein,
                        shall have the meaning given to them in the Service
                        Agreement. Except as modified or supplemented below, the
                        definitions of the Service Agreement, as well as all the
                        other terms and conditions of the Service Agreement,
                        shall remain in full force and effect.
                      </Text>{' '}
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        For the purpose of interpreting this Addendum, the
                        following terms shall have the meanings set out below:
                      </Text>{' '}
                      <List type="ordered" listStyleType="lower-roman">
                        <List.Item>
                          <Text size="md" mb="md" c="black">
                            “
                            <Text span size="md" fw={600}>
                              Applicable Laws
                            </Text>
                            ” means all laws applicable to the Processing of
                            Client Personal Data, including EU Data Protection
                            Laws, other laws of the European Union or any Member
                            State thereof, the UK GDPR and the UK Data
                            Protection Act 2018, the Singapore Personal Data
                            Protection Act 2012, and the laws of any other
                            country to which the Processing of Client Personal
                            Data is subject;
                          </Text>
                        </List.Item>
                        <List.Item>
                          <Text size="md" mb="md" c="black">
                            “
                            <Text span size="md" fw={600}>
                              Client
                            </Text>
                            ” means the party that has entered into this
                            Addendum with Cosmo, as indicated in the opening
                            paragraph of this Addendum, including all affiliates
                            of that entity that are also bound by the Service
                            Agreement, if any;
                          </Text>
                        </List.Item>
                        <List.Item>
                          <Text size="md" mb="md" c="black">
                            “
                            <Text span size="md" fw={600}>
                              Client Personal Data
                            </Text>
                            ” means any Personal Data Processed by Cosmo or a
                            Subprocessor on behalf of the Client pursuant to or
                            in connection with the Service Agreement;
                          </Text>
                        </List.Item>
                        <List.Item>
                          <Text size="md" mb="md" c="black">
                            “
                            <Text span size="md" fw={600}>
                              Contracted Processor
                            </Text>
                            ” means Cosmo, a Subprocessor, or both collectively;
                          </Text>
                        </List.Item>
                        <List.Item>
                          <Text size="md" mb="md" c="black">
                            “
                            <Text span size="md" fw={600}>
                              EU Data Protection Laws
                            </Text>
                            ” means the GDPR, the domestic legislation of each
                            Member State implementing and supplementing the
                            GDPR, as well as other laws of the European Union or
                            any Member State thereof to which the Processing of
                            Client Personal Data is subject, as amended,
                            replaced, or superseded from time to time;
                          </Text>
                        </List.Item>
                        <List.Item>
                          <Text size="md" mb="md" c="black">
                            “
                            <Text span size="md" fw={600}>
                              GDPR
                            </Text>
                            ” means Regulation (EU) 2016/679 of the European
                            Parliament and of the Council of 27 April 2016 on
                            the Protection of Natural Persons with Regard to the
                            Processing of Personal Data and on the Free Movement
                            of Such Data, and Repealing Directive 95/46/EC
                            (General Data Protection Regulation);
                          </Text>
                        </List.Item>
                        <List.Item>
                          <Text size="md" mb="md" c="black">
                            “
                            <Text span size="md" fw={600}>
                              Restricted Transfer
                            </Text>
                            ” means any transfer of Client Personal Data that
                            would be prohibited by EU Data Protection Laws (or
                            by the terms of data transfer agreements put in
                            place to address the data transfer restrictions of
                            EU Data Protection Laws) in the absence of the
                            execution of the Standard Contractual Clauses or
                            another lawful data transfer mechanism, as set out
                            in Section 12 below;
                          </Text>
                        </List.Item>
                        <List.Item>
                          <Text size="md" mb="md" c="black">
                            “
                            <Text span size="md" fw={600}>
                              Services
                            </Text>
                            ” means the services and other activities to be
                            supplied to or carried out by or on behalf of Cosmo
                            for the Client pursuant to the Service Agreement;
                            and
                          </Text>
                        </List.Item>
                        <List.Item>
                          <Text size="md" mb="md" c="black">
                            “
                            <Text span size="md" fw={600}>
                              Subprocessor
                            </Text>
                            ” means any natural or legal person (including any
                            third party but excluding an employee of Cosmo or an
                            employee of any of its sub-contractors) appointed by
                            or on behalf of Cosmo to Process Client Personal
                            Data on behalf of the Client in connection with the
                            Service Agreement.
                          </Text>
                        </List.Item>
                      </List>
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        The terms
                        <Text span size="md" fw={600}>
                          “Controller”, “Data Subject”, “Member State”,
                          “Personal Data”, “Personal Data Breach”, “Processing”,
                          “Processor”, “Rights of the Data Subjects”,
                          “Supervisory Authority”, and “Third Country”
                        </Text>
                        whether capitalized or not, shall have the same meaning
                        as in the GDPR, and their cognate terms shall be
                        construed accordingly.
                      </Text>
                    </List.Item>
                  </List>
                </List.Item>
                {/* 2 */}
                <List.Item c="blue">
                  <Text size="md" mb="md" c="black" fw={600}>
                    Applicability.
                  </Text>
                  <List type="ordered" listStyleType="lower-alpha">
                    {' '}
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        This Addendum applies to the Processing of Client
                        Personal Data that is regulated by (i) EU Data
                        Protection Laws, (ii) the UK GDPR and the UK Data
                        Protection Act 2018, or (iii) the Singapore Personal
                        Data Protection Act 2012 (the “PDPA”). Where the
                        operating entity is established in Singapore,
                        Processing carried out by Cosmo will accordingly also
                        be subject to the PDPA. References in this Addendum to EU Data Protection
                        Laws shall, where the relevant Processing is regulated
                        by the UK GDPR or the PDPA, be read as references to
                        those laws with the changes necessary to give them
                        effect, and references to Articles of the GDPR shall be
                        read as references to the corresponding provisions of
                        the UK GDPR. Where the Processing of Client Personal
                        Data is regulated by data protection laws other than
                        those identified in this Section 2.1, the Parties shall
                        negotiate in good faith and agree such additional terms
                        as those laws require. The Parties to this Addendum
                        hereby agree that the terms and conditions set out
                        herein shall be added as an addendum to the Service
                        Agreement. Except where the context requires otherwise,
                        references in this Addendum to the Service Agreement are
                        to the Service Agreement as amended or supplemented by,
                        and including, this Addendum.
                      </Text>{' '}
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        The Terms of this Addendum shall take effect on the
                        Effective Date and shall continue concurrently for the
                        term of the Service Agreement.
                      </Text>{' '}
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        If you enter into this Addendum, or enter into a Service
                        Agreement that incorporates this Addendum by reference,
                        on behalf of an entity, you represent and warrant to
                        Cosmo that you have the authority to bind that entity
                        and its affiliates, where applicable, to the terms and
                        conditions of this Addendum.
                      </Text>{' '}
                    </List.Item>
                  </List>
                </List.Item>
                {/* 3 */}
                <List.Item c="blue">
                  <Text size="md" mb="md" c="black" fw={600}>
                    Processing of Client Personal Data.
                  </Text>
                  <List type="ordered" listStyleType="lower-alpha">
                    {' '}
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        In the context of this Addendum, the Client acts as a
                        Personal Data Controller and Cosmo acts as a Personal
                        Data Processor with regard to the Processing of Client
                        Personal Data.
                      </Text>{' '}
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        Cosmo shall:
                      </Text>{' '}
                      <List type="ordered" listStyleType="lower-roman">
                        <List.Item>
                          <Text size="md" mb="md" c="black">
                            comply with all EU Data Protection Laws in the
                            Processing (as further elaborated in{' '}
                            <Text
                              span
                              size="md"
                              mb="md"
                              td="underline"
                              c="black"
                              fw={600}
                            >
                              Exhibit A
                            </Text>
                            , attached hereto and incorporated by reference) of
                            Client Personal Data;
                          </Text>{' '}
                        </List.Item>
                        <List.Item>
                          <Text size="md" mb="md" c="black">
                            not Process Client Personal Data other than on the
                            Client’s relevant documented instructions, including
                            with regard to transfers of Client Personal Data to
                            a Third Country or an international organization,
                            unless such Processing is required by EU Data
                            Protection Laws to which the relevant Contracted
                            Processor is subject, in which case Cosmo shall, to
                            the extent permitted by EU Data Protection Laws,
                            inform the Client of that legal requirement before
                            the applicable act of Processing;
                          </Text>{' '}
                        </List.Item>
                        <List.Item>
                          <Text size="md" mb="md" c="black">
                            only conduct transfers of Client Personal Data,
                            where such transfer would otherwise be prohibited by
                            EU Data Protection Laws due to there being no
                            applicable lawful exemption or derogation, in
                            compliance with all applicable conditions, as laid
                            down in the EU Data Protection Laws;
                          </Text>{' '}
                        </List.Item>
                        <List.Item>
                          <Text size="md" mb="md" c="black">
                            not retain, delete, or otherwise Process Client
                            Personal Data contrary to or in the absence of the
                            direct instructions of the Client, provided,
                            however, that the Client expressly and irrevocably
                            authorizes such retention, deletion or other
                            Processing if and to the extent required or allowed
                            by any applicable law; and
                          </Text>{' '}
                        </List.Item>
                        <List.Item>
                          <Text size="md" mb="md" c="black">
                            immediately inform the Client in the event that, in
                            Cosmo’s opinion, a Processing instruction given by
                            the Client may infringe EU Data Protection Laws.
                          </Text>{' '}
                        </List.Item>
                      </List>
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        The Client warrants that it will promptly update, when
                        necessary, all information provided in the course of
                        entering into this Addendum, including, where
                        applicable, the contact details of its Data Protection
                        Officer and/or European Union Representative. Any such
                        updates shall be sent by email to{' '}
                        <Text size="md" mb="md" c="blue" td="underline" span>
                          dpo@cosmoagents.ai.
                        </Text>{' '}
                      </Text>{' '}
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        The Client instructs Cosmo (and authorizes Cosmo to
                        instruct each Subprocessor) to Process Client Personal
                        Data, and in particular, transfer Client Personal Data
                        to any country or territory, as reasonably necessary for
                        the provision of the Services and consistent with the
                        Service Agreement and this Addendum.
                      </Text>{' '}
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        The Client represents and warrants that it has all
                        necessary rights to provide the Client Personal Data to
                        Cosmo for the purpose of Processing such data within the
                        scope of this Addendum and the Service Agreement.
                      </Text>{' '}
                    </List.Item>
                  </List>
                </List.Item>
                {/* 4 */}
                <List.Item c="blue">
                  <Text size="md" mb="md" c="black" fw={600}>
                    Cosmo Personnel.
                  </Text>
                  <List type="ordered" listStyleType="lower-alpha">
                    {' '}
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        Cosmo shall take reasonable steps to ensure the
                        reliability of any employee, agent, or contractor of any
                        Contracted Processor who may have access to the Client
                        Personal Data, ensuring in each case that access is
                        strictly limited to those individuals who need to know
                        or access the relevant Client Personal Data, as strictly
                        necessary for the purposes of the Service Agreement, and
                        to comply with EU Data Protection Laws in the context of
                        that individual’s duties to the Contracted Processor,
                        ensuring that all such individuals are subject to formal
                        confidentiality undertakings or professional or
                        statutory obligations of confidentiality.
                      </Text>{' '}
                    </List.Item>
                  </List>
                </List.Item>
                {/* 5 */}
                <List.Item c="blue">
                  <Text size="md" mb="md" c="black" fw={600}>
                    Security of Processing.
                  </Text>
                  <List type="ordered" listStyleType="lower-alpha">
                    {' '}
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        Taking into account the state of the art, the costs of
                        implementation and the nature, scope, context, and
                        purposes of Processing, as well as the risk of varying
                        likelihood and severity to the rights and freedoms of
                        natural persons, Cosmo shall, with regard to Client
                        Personal Data, implement and maintain{' '}
                        <Text size="md" c="blue" span>
                          appropriate technical and organizational security
                          measures
                        </Text>{' '}
                        to ensure a level of security appropriate to that risk,
                        having regard to the measures referred to in Article
                        32(1) of the GDPR. The technical and organizational
                        measures that are in place as of the Effective Date, and
                        those that Cosmo has committed to implement and the
                        horizon for doing so, are described in Appendix 2 to the
                        Standard Contractual Clauses set out in{' '}
                        <Text
                          span
                          size="md"
                          mb="md"
                          td="underline"
                          c="black"
                          fw={600}
                        >
                          Exhibit C
                        </Text>
                        . Cosmo shall not materially reduce the overall level of
                        security described in Appendix 2 during the term of the
                        Service Agreement, and shall assist the Client with
                        regard to ensuring compliance with the obligations
                        pursuant to Article 32 of the GDPR borne directly by the
                        Client.
                      </Text>{' '}
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        In assessing the appropriate level of security, Cosmo
                        shall take account, in particular, of the risks that are
                        presented by the nature of such Processing activities,
                        and particularly those related to possible Personal Data
                        Breaches.
                      </Text>{' '}
                    </List.Item>
                  </List>
                </List.Item>
                {/* 6 */}
                <List.Item c="blue">
                  <Text size="md" mb="md" c="black" fw={600}>
                    Subprocessing.
                  </Text>
                  <List type="ordered" listStyleType="lower-alpha">
                    {' '}
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        The Client authorizes Cosmo to appoint (and permit each
                        Subprocessor appointed in accordance with this Section 6
                        to appoint) Subprocessors in accordance with this
                        Section 6 and any possible further restrictions, as set
                        out in the Service Agreement and this Addendum.
                      </Text>{' '}
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        Cosmo may continue to use those Subprocessors already
                        engaged by Cosmo as of the Effective Date subject to
                        Cosmo meeting the obligations set out in Section 6.4.
                        The list of Cosmo’s Subprocessors, current as of the
                        Effective Date, is laid down in{' '}
                        <Text size="md" mb="md" c="black" span fw={600}>
                          Exhibit B
                        </Text>{' '}
                        , attached hereto and incorporated by reference.
                      </Text>{' '}
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        Cosmo shall give the Client prior written notice of the
                        appointment of any new Subprocessor, by way of sending a
                        notice. If, within 14 days of sending of each such
                        notice, the Client notifies Cosmo in writing of any
                        reasonable objections to the proposed appointment, Cosmo
                        shall not appoint or disclose any Client Personal Data
                        to that proposed Subprocessor until reasonable steps
                        have been taken to address the objections raised by the
                        Client and, in turn, the Client has been provided with a
                        reasonable written explanation of the steps taken to
                        account for any such objections. If the Client,
                        nevertheless, objects to the proposed appointment, it
                        shall be entitled to terminate the Service Agreement as
                        a remedy.
                      </Text>{' '}
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        With respect to each Subprocessor, Cosmo shall:
                      </Text>{' '}
                      <List type="ordered" listStyleType="lower-roman">
                        <List.Item>
                          <Text size="md" mb="md" c="black">
                            carry out adequate due diligence to ensure that the
                            Subprocessor is capable of providing the level of
                            protection for Client Personal Data required by this
                            Addendum, the Service Agreement, and EU Data
                            Protection Laws before the Subprocessor first
                            Processes Client Personal Data or, where applicable,
                            in accordance with Section 6.2; and
                          </Text>{' '}
                        </List.Item>
                        <List.Item>
                          <Text size="md" mb="md" c="black">
                            ensure that the arrangement between: on the one
                            hand, (i) Cosmo, or (ii) the relevant intermediate
                            Subprocessor; and on the other hand, the respective
                            prospective Subprocessor, is governed by a written
                            contract, including terms which offer at least the
                            same level of protection for Client Personal Data as
                            those set out in this Addendum, and that such terms
                            meet the requirements of Article 28(3) of the GDPR.
                          </Text>{' '}
                        </List.Item>
                      </List>
                    </List.Item>
                  </List>
                </List.Item>
                {/* 7 */}
                <List.Item c="blue">
                  <Text size="md" mb="md" c="black" fw={600}>
                    Rights of the Data Subjects.
                  </Text>
                  <List type="ordered" listStyleType="lower-alpha">
                    {' '}
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        Taking into account the nature of the Processing, Cosmo
                        shall assist the Client by implementing appropriate
                        technical and organizational measures, insofar as this
                        is possible, for the fulfilment of the Client’s
                        obligations, as reasonably understood by the Client, to
                        respond to requests to exercise Rights of the Data
                        Subjects under the EU Data Protection Laws.
                      </Text>{' '}
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        With regard to Rights of the Data Subjects within the
                        scope of this Section 7, Cosmo shall:
                      </Text>{' '}
                      <List type="ordered" listStyleType="lower-roman">
                        <List.Item>
                          <Text size="md" mb="md" c="black">
                            promptly notify the Client if any Contracted
                            Processor receives a request from a Data Subject
                            under any EU Data Protection Laws in respect of
                            Client Personal Data; and
                          </Text>{' '}
                        </List.Item>
                        <List.Item>
                          <Text size="md" mb="md" c="black">
                            ensure that the Contracted Processor does not
                            respond to that request, except on the documented
                            instructions of the Client, or as required by EU
                            Data Protection Laws to which the Contracted
                            Processor is subject, in which case Cosmo shall, to
                            the extent permitted by EU Data Protection Laws,
                            inform the Client of that legal requirement before
                            the Contracted Processor responds to the request.
                          </Text>{' '}
                        </List.Item>
                      </List>
                    </List.Item>
                  </List>
                </List.Item>
                {/* 8 */}
                <List.Item c="blue">
                  <Text size="md" mb="md" c="black" fw={600}>
                    Personal Data Breach.
                  </Text>
                  <List type="ordered" listStyleType="lower-alpha">
                    {' '}
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        Cosmo shall notify the Client without undue delay upon
                        Cosmo becoming aware of a Personal Data Breach affecting
                        Client Personal Data under Cosmo’s direct control or
                        upon Cosmo being notified of a Personal Data Breach
                        affecting Client Personal Data under the direct control
                        of a Subprocessor, providing the Client with sufficient
                        information to allow the Client to meet any applicable
                        obligations pursuant to the EU Data Protection Laws,
                        such as to report to the Supervisory Authorities or any
                        other competent authorities, or inform the Data Subjects
                        of the Personal Data Breach.
                      </Text>{' '}
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        Cosmo shall co-operate with the Client and take all
                        reasonable commercial steps to assist the Client in the
                        investigation, mitigation, and remediation of each such
                        Personal Data Breach.
                      </Text>{' '}
                    </List.Item>
                  </List>
                </List.Item>
                {/* 9 */}
                <List.Item c="blue">
                  <Text size="md" mb="md" c="black" fw={600}>
                    Data Protection Impact Assessment and Prior Consultation.
                  </Text>
                  <List type="ordered" listStyleType="lower-alpha">
                    {' '}
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        Cosmo shall provide the Client with relevant
                        documentation, such as, if available, an audit report
                        (upon a written request and subject to obligations of
                        confidentiality), with regard to any data protection
                        impact assessments, and prior consultations with
                        Supervisory Authorities or other competent data privacy
                        authorities, when the Client reasonably considers that
                        such data protection impact assessments or prior
                        consultations are required pursuant to Article 35 or 36
                        of the GDPR, or pursuant to the equivalent provisions of
                        any other EU Data Protection Laws but, in each such
                        case, solely with regard to Processing of Client
                        Personal Data by, and taking into account the nature of
                        the Processing and information available to, the
                        respective Contracted Processors.
                      </Text>{' '}
                    </List.Item>
                  </List>
                </List.Item>
                {/* 10 */}
                <List.Item c="blue">
                  <Text size="md" mb="md" c="black" fw={600}>
                    Deletion or Return of Client Personal Data.
                  </Text>
                  <List type="ordered" listStyleType="lower-alpha">
                    {' '}
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        Cosmo shall provide the Client with the technical means,
                        consistent with the way the Services are provided, to
                        request the deletion of Client Personal Data within the
                        term of this Addendum and the Service Agreement, unless
                        Applicable Laws require or allow storage of any such
                        Client Personal Data. A deletion request may also be
                        made in writing to{' '}
                        <Text size="md" mb="md" c="blue" td="underline" span>
                          dpo@cosmoagents.ai
                        </Text>
                        . Cosmo shall acknowledge each such request within five
                        (5) business days of receipt and shall action it in
                        accordance with Section 10.2.
                      </Text>{' '}
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        On receipt of a deletion request, the relevant records
                        are first marked as deleted and withdrawn from active
                        Processing, so that they cease to be accessible through
                        the Services and are no longer used to provide them.
                        Cosmo shall then complete the erasure of those records
                        and of the data derived from them, including vector
                        embeddings, indexed knowledge content, and AI-generated
                        summaries, and shall instruct each Subprocessor listed
                        in Exhibit B that holds the corresponding data to do the
                        same, in each case within thirty (30) days of the
                        request. As of the Effective Date, the erasure step is
                        performed by Cosmo personnel on request rather than by
                        an automated process; automated retention windows and
                        scheduled purging of records marked as deleted, together
                        with the automated expiry of derived data held in the
                        vector store, are scheduled for implementation before
                        general availability of the Services.
                      </Text>{' '}
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        Following the date of cessation of the Services
                        involving the Processing of Client Personal Data, Cosmo
                        shall, at the choice of the Client, delete or return all
                        Client Personal Data to the Client, and delete existing
                        copies, in each case within thirty (30) days of that
                        date, unless Applicable Laws require or allow storage of
                        any such Client Personal Data. Where Applicable Laws
                        require continued storage, Cosmo shall inform the Client
                        of the requirement and shall Process the retained data
                        only for the purpose, and for the period, that those
                        laws require.
                      </Text>{' '}
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        Backup copies containing Client Personal Data are not
                        erased individually. They are overwritten in the
                        ordinary course of the backup rotation cycle and remain
                        subject to the terms of this Addendum, and to the
                        confidentiality and security obligations set out in
                        Section 4 and Section 5, until they are overwritten.
                      </Text>{' '}
                    </List.Item>
                  </List>
                </List.Item>
                {/* 11 */}
                <List.Item c="blue">
                  <Text size="md" mb="md" c="black" fw={600}>
                    Audit Rights.
                  </Text>
                  <List type="ordered" listStyleType="lower-alpha">
                    {' '}
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        Where the Client is entitled to and desires to review
                        Cosmo’s compliance with the EU Data Protection Laws, the
                        Client may request, and Cosmo will provide (subject to
                        obligations of confidentiality) relevant documentation,
                        or any relevant audit report Cosmo might have been
                        issued. If the Client, after having reviewed such audit
                        report(s), still reasonably deems that it requires
                        additional information, Cosmo shall further reasonably
                        assist and make available to the Client, upon a written
                        request and subject to obligations of confidentiality,
                        all other information (excluding legal advice) and/or
                        documentation necessary to demonstrate compliance with
                        this Addendum, and the obligations pursuant to Articles
                        32 to 36 of the GDPR in particular, and shall allow for
                        and contribute to audits, including remote inspections
                        of the Services, by the Client or an auditor mandated by
                        the Client with regard to the Processing of the Client
                        Personal Data by the Contracted Processors. Cosmo shall
                        provide the assistance described in this Section 11,
                        insofar as in Cosmo’s reasonable opinion such audits,
                        and the specific requests of the Client, do not
                        interfere with Cosmo’s business operations or cause
                        Cosmo to breach any legal or contractual obligation to
                        which it is subject.
                      </Text>{' '}
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        The Client agrees to pay Cosmo, upon receipt of invoice,
                        a reasonable fee based on the time spent, as well as to
                        account for the materials expended, in relation to the
                        Client exercising its rights under this Section 11 or
                        Clause 8.9 of the Standard Contractual Clauses, as set
                        out in{' '}
                        <Text size="md" fw={600} td="underline" span>
                          Exhibit C
                        </Text>{' '}
                        , attached hereto and incorporated by reference, and
                        which constitute an integral part of this Addendum (the
                        “
                        <Text size="md" fw={600} td="underline" span>
                          Standard Contractual Clauses
                        </Text>{' '}
                        ”).
                      </Text>{' '}
                    </List.Item>
                  </List>
                </List.Item>
                {/* 12 */}
                <List.Item c="blue">
                  <Text size="md" mb="md" c="black" fw={600}>
                    Restricted Transfers.
                  </Text>
                  <List type="ordered" listStyleType="lower-alpha">
                    {' '}
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        The Client (as “data exporter”) and Cosmo (as “data
                        importer”) hereby enter into, as of the Effective Date,
                        the Standard Contractual Clauses. The Parties are deemed
                        to have accepted and executed the Standard Contractual
                        Clauses in their entirety, including the appendices.
                      </Text>{' '}
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        With regard to any Restricted Transfer from the Client
                        to Cosmo within the scope of this Addendum, one of the
                        following transfer mechanisms shall apply, in the
                        following order of precedence:
                      </Text>{' '}
                      <List type="ordered" listStyleType="lower-roman">
                        <List.Item>
                          <Text size="md" mb="md" c="black">
                            the Standard Contractual Clauses, as set out in
                            Exhibit C, which shall be the primary transfer
                            mechanism for every Restricted Transfer from the
                            Client to Cosmo and which apply in every case in
                            which no adequacy decision covers the transfer;
                          </Text>{' '}
                        </List.Item>
                        <List.Item>
                          <Text size="md" mb="md" c="black">
                            where the European Commission has adopted a decision
                            under Article 45 of the GDPR (or the competent UK or
                            Singapore authority has made an equivalent finding)
                            that the country or territory of destination ensures
                            an adequate level of protection, and that decision
                            is in force and covers the transfer, that adequacy
                            decision; or
                          </Text>{' '}
                        </List.Item>
                        <List.Item>
                          <Text size="md" mb="md" c="black">
                            any other lawful transfer mechanism available under
                            Applicable Laws, including the derogations laid down
                            in Article 49 of the GDPR where they apply.
                          </Text>{' '}
                        </List.Item>
                      </List>
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        In cases where the Standard Contractual Clauses apply
                        and there is a conflict between the terms of the
                        Addendum and the terms of the Standard Contractual
                        Clauses, the terms of the Standard Contractual Clauses
                        shall control.
                      </Text>{' '}
                    </List.Item>
                  </List>
                </List.Item>
                {/* 13 */}
                <List.Item c="blue">
                  <Text size="md" mb="md" c="black" fw={600}>
                    General Terms.
                  </Text>
                  <List type="ordered" listStyleType="lower-alpha">
                    {' '}
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        All clauses of the Service Agreement that are not
                        explicitly amended or supplemented by the clauses of
                        this Addendum shall remain in full force and effect and
                        shall apply so long as they do not contradict Applicable
                        Laws.
                      </Text>{' '}
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        Cosmo may amend the terms of this Addendum, insofar as
                        the revised Addendum continues to comply with the
                        relevant requirements of the EU Data Protection Laws,
                        upon notice to the Client by email to the primary
                        contact on the account. Any such amendments will
                        automatically become effective 10 days after Cosmo’s
                        transmission of each such notice.
                      </Text>{' '}
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        In the event of any conflict between the Service
                        Agreement (including any annexes and appendices thereto)
                        and this Addendum, the provisions of this Addendum shall
                        control.
                      </Text>{' '}
                    </List.Item>
                  </List>
                </List.Item>
              </List>
              <Text size="md" mb="md">
                Should any provision of this Addendum be found invalid or
                unenforceable pursuant to any applicable law, then the invalid
                or unenforceable provision will be deemed superseded by a valid,
                enforceable provision that most closely matches the intent of
                the original provision and the remainder of the Addendum will
                continue in effect.
              </Text>
              <Text size="md" mb="md">
                If Cosmo makes a determination that it can no longer meet its
                obligations in accordance with this Addendum, it shall promptly
                notify the Client of that determination, and cease the
                Processing or take other reasonable and appropriate steps to
                remediate.
              </Text>
              {/* Data Protection Officer and EU Representative */}
              <Box>
                <Text size="md" mb="md" fw={600}>
                  Data Protection Officer and EU Representative.
                </Text>
                <Text size="md" mb="md">
                  Questions about this Addendum, requests relating to Rights of
                  the Data Subjects, and deletion requests may be addressed to
                  Cosmo’s data protection contact at{' '}
                  <Text size="md" mb="md" c="blue" td="underline" span>
                    dpo@cosmoagents.ai
                  </Text>
                  , or by post to Cosmo,
                  Singapore.
                </Text>
                <Text size="md" mb="md">
                  Where Article 27 of the GDPR or Article 27 of the UK GDPR
                  requires it, Cosmo will appoint a representative in the
                  European Union and in the United Kingdom respectively, and
                  will publish that representative’s contact details in this
                  Addendum before the appointment is required to be in place. No
                  such representative has been appointed as of the Effective
                  Date. Cosmo has not appointed a Data Protection Officer under
                  Article 37 of the GDPR and will appoint one if and when
                  Article 37 requires it; the address above is a contact point
                  and its use does not imply that such an appointment has been
                  made.
                </Text>
              </Box>

              <Title
                order={2}
                mt="lg"
                mb="md"
                id="exhibit-a"
                style={{ scrollMarginTop: '80px' }}
              >
                Exhibit A
              </Title>
              {/* Exhibit A Content */}
              <Box>
                <List type="ordered" c="blue">
                  <List.Item>
                    <Text size="md" mb="md" c="black">
                      Pursuant to Article 28(3) of the GDPR, further details of
                      the Processing, in addition to the ones laid down in the
                      Service Agreement and this Addendum, include:
                    </Text>
                    <Text size="md" mb="md" c="black">
                      The subject matter of the Processing of Client Personal
                      Data is:
                    </Text>
                    <Text size="md" mb="md" c="black">
                      The subject matter of the Processing of Client Personal
                      Data pertains to the provision of Services, as requested
                      by the Client.
                    </Text>
                  </List.Item>
                  <List.Item>
                    <Text size="md" mb="md" c="black">
                      The duration of the Processing of Client Personal Data is:
                    </Text>
                    <Text size="md" mb="md" c="black">
                      The duration of the Processing of Client Personal Data is
                      generally determined by the Client and is further subject
                      to the term of this Addendum and the Service Agreement,
                      respectively, in the context of the contractual
                      relationship between Cosmo and the Client.
                    </Text>
                  </List.Item>
                  <List.Item>
                    <Text size="md" mb="md" c="black">
                      The nature and purpose of the Processing of Client
                      Personal Data is:
                    </Text>
                    <Text size="md" mb="md" c="black">
                      The purpose of Processing of Client Personal Data pertains
                      to the provision of sales assistance, as requested by the
                      Client. The nature of such Processing is related to these
                      purposes and is elaborated on in this Addendum and the
                      Service Agreement.
                    </Text>
                  </List.Item>
                  <List.Item>
                    <Text size="md" mb="md" c="black">
                      The types of Client Personal Data to be Processed include:
                    </Text>
                    <Text size="md" mb="md" c="black">
                      The types of Client Personal Data Processed by Cosmo
                      comprise:
                    </Text>
                    <List c="black" mb="md">
                      <List.Item>
                        <Text size="md" c="black">
                          business contact details, including first and last
                          name, work email address, telephone number, job title,
                          seniority, employer or company name, and industry;
                        </Text>
                      </List.Item>
                      <List.Item>
                        <Text size="md" c="black">
                          postal address fields, including street address, city,
                          state or province, postal or zip code, and country;
                        </Text>
                      </List.Item>
                      <List.Item>
                        <Text size="md" c="black">
                          LinkedIn profile URLs and publicly available profile
                          information collected from those profiles, including
                          role history and profile summaries;
                        </Text>
                      </List.Item>
                      <List.Item>
                        <Text size="md" c="black">
                          interaction history and email content, including the
                          subject lines and bodies of messages sent and received
                          through the connected mailbox, the conversation
                          threads to which they belong, sender and recipient
                          details, timestamps, and engagement events such as
                          opens, replies, and bounces;
                        </Text>
                      </List.Item>
                      <List.Item>
                        <Text size="md" c="black">
                          notes, tags, and other free-text records entered about
                          a Data Subject by the Client’s personnel;
                        </Text>
                      </List.Item>
                      <List.Item>
                        <Text size="md" c="black">
                          enrichment data obtained from third-party data
                          providers, including firmographic and role information
                          about the Data Subject and the Data Subject’s
                          employer;
                        </Text>
                      </List.Item>
                      <List.Item>
                        <Text size="md" c="black">
                          data derived from the above by Cosmo or its
                          Subprocessors, including AI-generated summaries,
                          classifications, and vector embeddings; and
                        </Text>
                      </List.Item>
                      <List.Item>
                        <Text size="md" c="black">
                          metadata, and any other category of Personal Data that
                          the Client or a Data Subject includes in an email or
                          uploads to the Services.
                        </Text>
                      </List.Item>
                    </List>
                    <Text size="md" mb="md" c="black">
                      The Services are not designed for, and the Client shall
                      not submit, special categories of Personal Data within the
                      meaning of Article 9 of the GDPR or Personal Data relating
                      to criminal convictions and offences.
                    </Text>
                  </List.Item>
                  <List.Item>
                    <Text size="md" mb="md" c="black">
                      The categories of Data Subjects to whom the Client
                      Personal Data relates include:
                    </Text>
                    <Text size="md" mb="md" c="black">
                      Sales prospects of the Client (Cosmo’s customer).
                    </Text>
                  </List.Item>
                  <List.Item>
                    <Text size="md" mb="md" c="black">
                      The obligations and rights of the Client are:
                    </Text>
                    <Text size="md" mb="md" c="black">
                      The rights and obligations of the Client are set out in
                      the Service Agreement and this Addendum.
                    </Text>
                  </List.Item>
                </List>
              </Box>
              <Title
                order={2}
                mt="lg"
                mb="md"
                id="exhibit-b"
                style={{ scrollMarginTop: '80px' }}
              >
                Exhibit B
              </Title>
              {/* Exhibit B Content */}
              <Box>
                <Text size="md" mb="md" ta="center">
                  List of Subprocessors
                </Text>
                <Text size="md" mb="md">
                  Below is the list of the Subprocessors engaged by Cosmo,
                  current as of the Effective Date, pursuant to Section 6.2 of
                  the Addendum. Where the table states that a Subprocessor is
                  engaged only on the Client’s election, that Subprocessor
                  Processes Client Personal Data solely where the Client enables
                  the corresponding integration.
                </Text>
                <Table
                  withTableBorder
                  withColumnBorders
                  striped
                  mb="md"
                  verticalSpacing="sm"
                >
                  <Table.Thead>
                    <Table.Tr>
                      <Table.Th>Sub-processor</Table.Th>
                      <Table.Th>Purpose</Table.Th>
                      <Table.Th>Location</Table.Th>
                    </Table.Tr>
                  </Table.Thead>
                  <Table.Tbody>
                    <Table.Tr>
                      <Table.Td>Amazon Web Services, Inc.</Table.Td>
                      <Table.Td>
                        Cloud infrastructure hosting for the application
                        servers, database, object storage, queues, and backups.
                      </Table.Td>
                      <Table.Td>Singapore (AWS region ap-southeast-1)</Table.Td>
                    </Table.Tr>
                    <Table.Tr>
                      <Table.Td>Google LLC</Table.Td>
                      <Table.Td>
                        Gmail API and Cloud Pub/Sub: sending, reading, and
                        labelling messages in the Client user’s connected Google
                        mailbox and receiving mailbox change notifications.
                        Engaged only where the Client user connects a Google
                        mailbox.
                      </Table.Td>
                      <Table.Td>United States</Table.Td>
                    </Table.Tr>
                    <Table.Tr>
                      <Table.Td>OpenAI, L.L.C.</Table.Td>
                      <Table.Td>
                        Large language model inference and generation of text
                        embeddings used for search, classification, and message
                        personalization.
                      </Table.Td>
                      <Table.Td>United States</Table.Td>
                    </Table.Tr>
                    <Table.Tr>
                      <Table.Td>Anthropic PBC</Table.Td>
                      <Table.Td>
                        Large language model inference for the in-app assistant.
                      </Table.Td>
                      <Table.Td>United States</Table.Td>
                    </Table.Tr>
                    <Table.Tr>
                      <Table.Td>Coze (ByteDance Ltd.)</Table.Td>
                      <Table.Td>
                        AI agent platform powering the in-app message writer and
                        reply drafting, and hosting the knowledge datasets those
                        agents query.
                      </Table.Td>
                      <Table.Td>United States</Table.Td>
                    </Table.Tr>
                    <Table.Tr>
                      <Table.Td>Apollo.io</Table.Td>
                      <Table.Td>
                        Prospect data enrichment and the supply of business
                        contact records.
                      </Table.Td>
                      <Table.Td>United States</Table.Td>
                    </Table.Tr>
                    <Table.Tr>
                      <Table.Td>Resend, Inc.</Table.Td>
                      <Table.Td>
                        Transactional and system email delivery.
                      </Table.Td>
                      <Table.Td>United States</Table.Td>
                    </Table.Tr>
                    <Table.Tr>
                      <Table.Td>HubSpot, Inc.</Table.Td>
                      <Table.Td>
                        Optional CRM synchronization. Engaged only where the
                        Client connects a HubSpot account.
                      </Table.Td>
                      <Table.Td>United States</Table.Td>
                    </Table.Tr>
                    <Table.Tr>
                      <Table.Td>Microsoft Corporation</Table.Td>
                      <Table.Td>
                        Optional Outlook and Microsoft 365 mailbox integration
                        via Microsoft Graph. Engaged only where the Client user
                        connects an Outlook mailbox.
                      </Table.Td>
                      <Table.Td>United States</Table.Td>
                    </Table.Tr>
                    <Table.Tr>
                      <Table.Td>Sentry</Table.Td>
                      <Table.Td>
                        Application error monitoring, including session replay
                        of the Client user’s interactions with the web
                        application.
                      </Table.Td>
                      <Table.Td>United States</Table.Td>
                    </Table.Tr>
                  </Table.Tbody>
                </Table>
                <Text size="md" mb="md">
                  Cosmo is the contracting party under this
                  Addendum and is therefore not listed as a Subprocessor.
                  Changes to this list are notified in accordance with Section
                  6.3 of the Addendum.
                </Text>
              </Box>

              <Title
                order={2}
                mt="lg"
                mb="md"
                id="exhibit-c"
                style={{ scrollMarginTop: '80px' }}
              >
                Exhibit C
              </Title>
              {/* Exhibit C Content */}
              <Box>
                <Text size="md" ta="center" fw={600}>
                  Commission Implementing Decision (EU) 2021/914 of 4 June 2021
                </Text>
                <Text size="md" mb="md" ta="center" fw={600}>
                  Standard Contractual Clauses – Module Two (Controller to
                  Processor)
                </Text>
                <Text size="md" mb="md">
                  For the purposes of Article 46(2)(c) of the GDPR, for the
                  transfer of personal data to processors established in third
                  countries that are not the subject of an adequacy decision
                  under Article 45 of the GDPR,
                </Text>
                <Text size="md" mb="md">
                  the Client, as defined in the Addendum (as “data exporter”),
                </Text>
                <Text size="md" mb="md">
                  and Cosmo, as defined in the Addendum (as “data importer”),
                  each a “party”; together “the parties”,
                </Text>
                <Text size="md" mb="md">
                  HAVE AGREED to the standard contractual clauses set out in the
                  Annex to Commission Implementing Decision (EU) 2021/914 of 4
                  June 2021 on standard contractual clauses for the transfer of
                  personal data to third countries pursuant to Regulation (EU)
                  2016/679 (the “Standard Contractual Clauses”), in order to
                  adduce adequate safeguards with respect to the protection of
                  privacy and fundamental rights and freedoms of individuals for
                  the transfer by the data exporter to the data importer of the
                  personal data described in Appendix 1 below.
                </Text>
                <Text size="md" mb="md">
                  The Standard Contractual Clauses are incorporated into this
                  Addendum by reference and form an integral part of it. The
                  full text is published in the Official Journal of the European
                  Union (OJ L 199, 7.6.2021, p. 31) and a copy will be provided
                  by Cosmo on request. The Standard Contractual Clauses are not
                  reproduced here; nothing in this Exhibit C varies them, and in
                  the event of any conflict between this Addendum and the
                  Standard Contractual Clauses, the Standard Contractual Clauses
                  prevail.
                </Text>
                <Text size="md" mb="md" fw={600}>
                  Modules and options selected
                </Text>
                <Text size="md" mb="md">
                  The parties agree that the following modules, options, and
                  specifications apply to the Standard Contractual Clauses as
                  incorporated by this Exhibit C:
                </Text>
                <List type="ordered" listStyleType="lower-alpha" mb="md">
                  <List.Item>
                    <Text size="md" c="black">
                      Module Two (transfer controller to processor) applies. The
                      Client is the data exporter and controller; Cosmo is the
                      data importer and processor. No other module applies.
                    </Text>
                  </List.Item>
                  <List.Item>
                    <Text size="md" c="black">
                      Clause 7 (the optional docking clause) applies.
                    </Text>
                  </List.Item>
                  <List.Item>
                    <Text size="md" c="black">
                      In Clause 9, Option 2 (general written authorisation)
                      applies. The time period for prior notice of Subprocessor
                      changes is the period specified in Section 6.3 of the
                      Addendum.
                    </Text>
                  </List.Item>
                  <List.Item>
                    <Text size="md" c="black">
                      In Clause 11, the optional independent dispute resolution
                      body wording does not apply.
                    </Text>
                  </List.Item>
                  <List.Item>
                    <Text size="md" c="black">
                      In Clause 13 and Annex I.C, the competent supervisory
                      authority is identified in Appendix 1 below.
                    </Text>
                  </List.Item>
                  <List.Item>
                    <Text size="md" c="black">
                      In Clause 17, Option 1 applies and the governing law is
                      the law of the Member State selected before commercial release. In Clause 18(b), the courts of the Member State selected under Clause 17
                      are the chosen forum. Where the transfer is subject to the
                      law of an EU Member State that allows for third-party
                      beneficiary rights, the parties may instead agree in the
                      Service Agreement on the law and forum of that Member
                      State.
                    </Text>
                  </List.Item>
                  <List.Item>
                    <Text size="md" c="black">
                      Annexes I, II, and III to the Standard Contractual Clauses
                      are populated as follows: Annex I (list of parties,
                      description of the transfer, and competent supervisory
                      authority) by Exhibit A of the Addendum and Appendix 1
                      below; Annex II (technical and organisational measures) by
                      Appendix 2 below; and Annex III (list of sub-processors)
                      by Exhibit B of the Addendum.
                    </Text>
                  </List.Item>
                </List>
                <Text size="md" mb="md" fw={600}>
                  Transfers subject to the UK GDPR
                </Text>
                <Text size="md" mb="md">
                  For any Restricted Transfer that is subject to the UK GDPR,
                  the Standard Contractual Clauses apply as varied by the
                  International Data Transfer Addendum to the EU Commission
                  Standard Contractual Clauses issued by the UK Information
                  Commissioner under section 119A of the UK Data Protection Act
                  2018, version B1.0 in force 21 March 2022 (the “UK Addendum”),
                  which is likewise incorporated by reference and which the
                  parties are deemed to have executed. Table 1 of the UK
                  Addendum is populated by the details of the parties set out in
                  Appendix 1 below; Table 2 identifies the Standard Contractual
                  Clauses described in this Exhibit C; Table 3 is populated by
                  Exhibits A and B of the Addendum and Appendix 2 below; and in
                  Table 4 neither party may end the UK Addendum as set out in
                  Section 19 of the UK Addendum. Where the UK Addendum applies,
                  references in the Standard Contractual Clauses to the GDPR, to
                  Member States, and to supervisory authorities are read in
                  accordance with the UK Addendum.
                </Text>
                <Text size="md" mb="md" fw={600}>
                  Transfers subject to Swiss law
                </Text>
                <Text size="md" mb="md">
                  For any Restricted Transfer that is subject to the Swiss
                  Federal Act on Data Protection (the “FADP”), the Standard
                  Contractual Clauses apply with the amendments recognised by
                  the Swiss Federal Data Protection and Information Commissioner
                  (the “FDPIC”), namely: references to the GDPR are read as
                  references to the FADP; the FDPIC is the competent supervisory
                  authority under Clause 13 and Annex I.C in respect of
                  transfers governed exclusively by the FADP; the term “Member
                  State” shall not be interpreted so as to prevent data subjects
                  in Switzerland from bringing proceedings in their place of
                  habitual residence in accordance with Clause 18(c); and, for
                  as long as the FADP so provides, the Standard Contractual
                  Clauses also protect the data of legal entities until the
                  entry into force of the revised FADP provisions.
                </Text>
                <Text size="md" mb="md" td="underline" ta="center" fw={600}>
                  Appendix 1 to the Standard Contractual Clauses (Annex I)
                </Text>
                <Text size="md" mb="md" td="underline" fw={600}>
                  By entering into the Standard Contractual Clauses, pursuant to
                  Section 12.1 of the Addendum, the parties are deemed to have
                  signed this Appendix 1.
                </Text>
                <Text size="md" mb="md" fw={600}>
                  Data exporter
                </Text>
                <Text size="md" mb="md">
                  The data exporter is the Client, as defined in the Addendum,
                  acting as controller. The Client’s contact details, and those
                  of its data protection officer or representative where one has
                  been designated, are the details provided by the Client under
                  Section 3.3 of the Addendum. The activities relevant to the
                  data transferred are the Client’s use of the Services.
                </Text>
                <Text size="md" mb="md" fw={600}>
                  Data importer
                </Text>
                <Text size="md" mb="md">
                  The data importer is Cosmo, as defined in the Addendum, being
                  Cosmo, a company incorporated
                  in Singapore, acting as processor. Contact point:
                  dpo@cosmoagents.ai. The activities relevant to the data
                  transferred are the provision of the Services described in
                  Section 3 of Exhibit A of the Addendum.
                </Text>
                <Text size="md" mb="md" fw={600}>
                  Data subjects
                </Text>
                <Text size="md" mb="md">
                  As indicated under Section 5 of Exhibit A of the Addendum.
                </Text>
                <Text size="md" mb="md" fw={600}>
                  Categories of personal data
                </Text>
                <Text size="md" mb="md">
                  As indicated under Section 4 of Exhibit A of the Addendum. No
                  special categories of personal data are transferred.
                </Text>
                <Text size="md" mb="md" fw={600}>
                  Nature and purpose of the processing
                </Text>
                <Text size="md" mb="md">
                  As indicated under Sections 1 and 3 of Exhibit A of the
                  Addendum.
                </Text>
                <Text size="md" mb="md" fw={600}>
                  Frequency of the transfer
                </Text>
                <Text size="md" mb="md">
                  Continuous, for the duration of the Service Agreement.
                </Text>
                <Text size="md" mb="md" fw={600}>
                  Duration of the processing and retention period
                </Text>
                <Text size="md" mb="md">
                  As indicated under Section 2 of Exhibit A and Section 10 of
                  the Addendum.
                </Text>
                <Text size="md" mb="md" fw={600}>
                  Transfers to sub-processors
                </Text>
                <Text size="md" mb="md">
                  The subject matter, nature, and duration of the processing
                  carried out by each Subprocessor are set out in Exhibit B of
                  the Addendum; each Subprocessor Processes the Client Personal
                  Data for the purpose stated against its name, for the duration
                  of the Service Agreement.
                </Text>
                <Text size="md" mb="md" fw={600}>
                  Competent supervisory authority
                </Text>
                <Text size="md" mb="md">
                  The supervisory authority of the EU Member State in which the
                  data exporter is established or, where the data exporter is
                  not established in the European Economic Area, the supervisory
                  authority of the Member State in which the data exporter’s
                  representative under Article 27 of the GDPR is established or,
                  in the absence of such a representative, the supervisory
                  authority of the Member State in which the data subjects whose
                  personal data is transferred are located. For transfers
                  governed exclusively by the FADP, the FDPIC. For transfers
                  governed by the UK GDPR, the UK Information Commissioner.
                </Text>
                <Text size="md" mb="md" td="underline" ta="center" fw={600}>
                  Appendix 2 to the Standard Contractual Clauses (Annex II)
                </Text>
                <Text size="md" mb="md" td="underline" fw={600}>
                  By entering into the Standard Contractual Clauses, pursuant to
                  Section 12.1 of the Addendum, the parties are deemed to have
                  signed this Appendix 2.
                </Text>
                <Text size="md" mb="md" fw={600}>
                  Description of the technical and organisational measures
                  implemented by the data importer to ensure an appropriate
                  level of security:
                </Text>
                <Text size="md" mb="md">
                  This Appendix 2 describes the measures that are in place as of
                  the Effective Date and, separately and expressly, the measures
                  that Cosmo has committed to implement and the horizon for
                  doing so. Cosmo makes no representation that a measure listed
                  as planned is in place. Cosmo holds no security certification
                  and has not undergone a SOC 2 examination, an ISO/IEC 27001
                  certification, or a third-party penetration test as of the
                  Effective Date.
                </Text>
                <Text size="md" mb="md" fw={600}>
                  Measures in place as of the Effective Date
                </Text>
                <List type="ordered" listStyleType="lower-alpha" mb="md">
                  <List.Item>
                    <Text size="md" c="black">
                      <Text span size="md" fw={600}>
                        Encryption of data in transit.
                      </Text>{' '}
                      All traffic between the Client’s browser, the Cosmo
                      application, and the Subprocessors listed in Exhibit B is
                      carried over TLS (HTTPS). Plain-text transport is not
                      offered.
                    </Text>
                  </List.Item>
                  <List.Item>
                    <Text size="md" c="black">
                      <Text span size="md" fw={600}>
                        Encryption of data at rest.
                      </Text>{' '}
                      Client Personal Data is stored on infrastructure operated
                      by Amazon Web Services in the ap-southeast-1 (Singapore)
                      region, using provider-managed disk and object storage
                      encryption at the infrastructure layer. Application-level
                      encryption of individual stored fields is not in place
                      today and is addressed under “Measures scheduled for
                      implementation” below.
                    </Text>
                  </List.Item>
                  <List.Item>
                    <Text size="md" c="black">
                      <Text span size="md" fw={600}>
                        Access control and tenant isolation.
                      </Text>{' '}
                      Client Personal Data is segregated logically. Every query
                      issued by the application is scoped to the authenticated
                      user and to that user’s organization, so that records
                      belonging to one Client are not returned to another.
                    </Text>
                  </List.Item>
                  <List.Item>
                    <Text size="md" c="black">
                      <Text span size="md" fw={600}>
                        Authentication.
                      </Text>{' '}
                      Access to the Services is authenticated using OAuth 2.0
                      and short-lived JSON Web Token access tokens. Mailbox
                      access is obtained by OAuth 2.0 authorisation granted by
                      the Client user and can be revoked by that user at any
                      time from the relevant provider’s account settings.
                    </Text>
                  </List.Item>
                  <List.Item>
                    <Text size="md" c="black">
                      <Text span size="md" fw={600}>
                        Least-privilege third-party scopes.
                      </Text>{' '}
                      Where a Client user connects a Google mailbox, Cosmo
                      requests only the scopes required to operate the Services,
                      namely gmail.send, gmail.readonly, gmail.modify, and
                      gmail.labels, together with Cloud Pub/Sub for mailbox
                      change notifications. Cosmo does not request access to
                      Google Drive or to any other Google service.
                    </Text>
                  </List.Item>
                  <List.Item>
                    <Text size="md" c="black">
                      <Text span size="md" fw={600}>
                        Secure development.
                      </Text>{' '}
                      Static security analysis of the server codebase (gosec) is
                      executed in the continuous integration pipeline on changes
                      to the codebase, and code changes are subject to peer
                      review before they are merged.
                    </Text>
                  </List.Item>
                  <List.Item>
                    <Text size="md" c="black">
                      <Text span size="md" fw={600}>
                        Logging and event monitoring.
                      </Text>{' '}
                      Application errors and exceptions are captured by the
                      error-monitoring Subprocessor identified in Exhibit B,
                      which supports the detection and investigation of
                      incidents, including Personal Data Breaches.
                    </Text>
                  </List.Item>
                  <List.Item>
                    <Text size="md" c="black">
                      <Text span size="md" fw={600}>
                        Personnel measures.
                      </Text>{' '}
                      Access to production systems is limited to those personnel
                      who require it in order to operate and support the
                      Services, and all such personnel are bound by
                      confidentiality obligations, as set out in Section 4 of
                      the Addendum.
                    </Text>
                  </List.Item>
                  <List.Item>
                    <Text size="md" c="black">
                      <Text span size="md" fw={600}>
                        Sub-processor governance.
                      </Text>{' '}
                      Each Subprocessor is engaged under a written contract on
                      terms that meet the requirements of Article 28(3) of the
                      GDPR, as set out in Section 6 of the Addendum, and the
                      current list is published in Exhibit B.
                    </Text>
                  </List.Item>
                </List>
                <Text size="md" mb="md" fw={600}>
                  Measures scheduled for implementation
                </Text>
                <Text size="md" mb="md">
                  The following measures are not in place as of the Effective
                  Date. Cosmo commits to implement each of them before the
                  general availability release of the Services, and will update
                  this Appendix 2 as each is completed.
                </Text>
                <List type="ordered" listStyleType="lower-alpha" mb="md">
                  <List.Item>
                    <Text size="md" c="black">
                      Application-level encryption of stored third-party
                      credentials, including OAuth access and refresh tokens,
                      under keys managed separately from the database.
                    </Text>
                  </List.Item>
                  <List.Item>
                    <Text size="md" c="black">
                      Pseudonymisation of Client Personal Data before it is
                      submitted to the artificial-intelligence Subprocessors
                      identified in Exhibit B, so that identifiers are replaced
                      before inference and restored on return.
                    </Text>
                  </List.Item>
                  <List.Item>
                    <Text size="md" c="black">
                      Automated retention windows and scheduled purging of
                      records marked as deleted, including the expiry of derived
                      data such as vector embeddings and AI-generated summaries,
                      as described in Section 10.2 of the Addendum.
                    </Text>
                  </List.Item>
                  <List.Item>
                    <Text size="md" c="black">
                      Single sign-on and multi-factor authentication for Client
                      accounts.
                    </Text>
                  </List.Item>
                  <List.Item>
                    <Text size="md" c="black">
                      A formal independent third-party security assessment of
                      the Services, the results of which will be made available
                      to the Client under Section 11 of the Addendum, subject to
                      obligations of confidentiality.
                    </Text>
                  </List.Item>
                </List>
              </Box>
            </Box>
          </AppShell.Main>
        </AppShell>
      </Box>
    </Box>
  );
}
