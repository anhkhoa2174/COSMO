'use client';

import { useState } from 'react';
import Link from 'next/link';
import NextImage from 'next/image';

import { Box, Group, Text, AppShell, NavLink, Title, List } from '@mantine/core';

export default function DataProcessing() {
  const [active, setActive] = useState<string>('');
  return (
    <Box>
      <Box>
        <AppShell
          header={{ height: 64 }}
          navbar={{
            width: 300,
            breakpoint: 'sm',
          }}
        >
          <AppShell.Header>
            <Group h="100%" p="md">
              <Group w="100%" align="center" gap="xs">
                <Link href="/">
                  <NextImage src="/favicon.svg" alt="LogoImg" width={42} height={42} />
                </Link>
                <Text tt="uppercase" c="#3C1988">
                  <Text span inherit fw={800}>
                    Cosmo
                  </Text>{' '}
                  <Text span inherit fw={500}>
                    Agents
                  </Text>
                </Text>
              </Group>
            </Group>
          </AppShell.Header>

          <AppShell.Navbar p="md">
            <NavLink
              href="#data-processing-addendum"
              label="Cosmo Agents DATA PROCESSING ADDENDUM"
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
                Cosmo Agents DATA PROCESSING ADDENDUM
              </Title>
              <Text size="md" mb="md">
                Controller to Processor
              </Text>
              <Text size="md" mb="md">
                This Cosmo Agents Data Processing Addendum (this “
                <Text span size="md" fw={600}>
                  Addendum
                </Text>
                ” ) is entered into by and between Cosmo Agents, Inc. (“
                <Text span size="md" fw={600}>
                  Cosmo Agents
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
                ”).This Addendum is effective as of the date you agree to it (the “
                <Text span size="md" fw={600}>
                  Effective Date
                </Text>
                ”) by clicking the “I Accept” button in the applicable online form or webpage that
                makes reference to this Addendum.
              </Text>

              <Text size="lg" mb="md" fw={600} py="lg" ta="center">
                RECITALS
              </Text>
              <Text size="md" mb="md">
                <Text span size="md" fw={600}>
                  WHEREAS
                </Text>
                , the Parties entered into the Cosmo Agents Services Order Form, which incorporates
                by reference the Cosmo Agents Terms of Use (collectively, the “
                <Text span size="md" fw={600}>
                  Service Agreement
                </Text>
                ”) and have retained the power to alter, amend, revoke, or terminate the Service
                Agreement; and
              </Text>
              <Text size="md" mb="md">
                <Text span size="md" fw={600}>
                  WHEREAS
                </Text>
                , the Parties now wish to amend the Service Agreement to ensure that Client Personal
                Data (as defined below) transferred between the Parties is Processed (as defined
                below) in compliance with applicable data protection principles and requirements;
              </Text>
              <Text size="md" mb="md">
                <Text span size="md" fw={600}>
                  NOW, THEREFORE
                </Text>
                , in consideration of the mutual agreements set forth in this Addendum, the Parties
                agree as follows:
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
                        The definitions used in this Addendum shall have the meanings set forth or
                        referenced in this Addendum. Capitalized definitions, not otherwise defined
                        herein, shall have the meaning given to them in the Service Agreement.
                        Except as modified or supplemented below, the definitions of the Service
                        Agreement, as well as all the other terms and conditions of the Service
                        Agreement, shall remain in full force and effect.
                      </Text>{' '}
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        For the purpose of interpreting this Addendum, the following terms shall
                        have the meanings set out below:
                      </Text>{' '}
                    </List.Item>
                  </List>
                </List.Item>
                {/* 2 */}
                <List.Item c="blue">
                  <Text size="md" mb="md" c="black">
                    “
                    <Text span size="md" fw={600}>
                      Applicable Laws
                    </Text>
                    ” means all laws applicable to the Processing of Client Personal Data, including
                    EU Data Protection Laws, other laws of the European Union or any Member State
                    thereof, and the laws of any other country to which the Processing of Client
                    Personal Data is subject;
                  </Text>
                </List.Item>
                {/* 3 */}
                <List.Item c="blue">
                  <Text size="md" mb="md" c="black">
                    “
                    <Text span size="md" fw={600}>
                      Client
                    </Text>
                    ” means the party that has entered into this Addendum with Cosmo Agents, as
                    indicated in the opening paragraph of this Addendum, including all affiliates of
                    that entity that are also bound by the Service Agreement, if any;
                  </Text>
                </List.Item>
                {/* 4 */}
                <List.Item c="blue">
                  <Text size="md" mb="md" c="black">
                    “
                    <Text span size="md" fw={600}>
                      Client Personal Data
                    </Text>
                    ” means any Personal Data Processed by Cosmo Agents or a Subprocessor on behalf
                    of the Client pursuant to or in connection with the Service Agreement;
                  </Text>
                </List.Item>
                {/* 5 */}
                <List.Item c="blue">
                  <Text size="md" mb="md" c="black">
                    “
                    <Text span size="md" fw={600}>
                      Contracted Processor
                    </Text>
                    ” means Cosmo Agents, a Subprocessor, or both collectively;
                  </Text>
                </List.Item>
                {/* 6 */}
                <List.Item c="blue">
                  <Text size="md" mb="md" c="black">
                    “
                    <Text span size="md" fw={600}>
                      EU Data Protection Laws
                    </Text>
                    ” means the GDPR, the domestic legislation of each Member State implementing and
                    supplementing the GDPR, as well as other laws of the European Union or any
                    Member State thereof to which the Processing of Client Personal Data is subject,
                    as amended, replaced, or superseded from time to time;
                  </Text>
                </List.Item>
                {/* 7 */}
                <List.Item c="blue">
                  <Text size="md" mb="md" c="black">
                    “
                    <Text span size="md" fw={600}>
                      GDPR
                    </Text>
                    ” means Regulation (EU) 2016/679 of the European Parliament and of the Council
                    of 27 April 2016 on the Protection of Natural Persons with Regard to the
                    Processing of Personal Data and on the Free Movement of Such Data, and Repealing
                    Directive 95/46/EC (General Data Protection Regulation);
                  </Text>
                </List.Item>
                {/* 8 */}
                <List.Item c="blue">
                  <Text size="md" mb="md" c="black">
                    “
                    <Text span size="md" fw={600}>
                      Restricted Transfer
                    </Text>
                    ” means any transfer of Client Personal Data that would be prohibited by EU Data
                    Protection Laws (or by the terms of data transfer agreements put in place to
                    address the data transfer restrictions of EU Data Protection Laws) in the
                    absence of the execution of the Standard Contractual Clauses or another lawful
                    data transfer mechanism, as set out in Section 12 below;
                  </Text>
                </List.Item>
                {/* 9 */}
                <List.Item c="blue">
                  <Text size="md" mb="md" c="black">
                    “
                    <Text span size="md" fw={600}>
                      Services
                    </Text>
                    ” means the services and other activities to be supplied to or carried out by or
                    on behalf of Cosmo Agents for the Client pursuant to the Service Agreement; and
                  </Text>
                </List.Item>
                {/* 10 */}
                <List.Item c="blue">
                  <Text size="md" mb="md" c="black">
                    “
                    <Text span size="md" fw={600}>
                      Subprocessor
                    </Text>
                    ” means any natural or legal person (including any third party but excluding an
                    employee of Cosmo Agents or an employee of any of its sub-contractors) appointed
                    by or on behalf of Cosmo Agents to Process Client Personal Data on behalf of the
                    Client in connection with the Service Agreement.
                  </Text>
                </List.Item>
                {/* 11 */}
                <List.Item c="blue">
                  <Text size="md" mb="md" c="black">
                    The terms
                    <Text span size="md" fw={600}>
                      “Controller”, “Data Subject”, “Member State”, “Personal Data”, “Personal Data
                      Breach”, “Processing”, “Processor”, “Rights of the Data Subjects”,
                      “Supervisory Authority”, and “Third Country”
                    </Text>
                    whether capitalized or not, shall have the same meaning as in the GDPR, and
                    their cognate terms shall be construed accordingly.
                  </Text>
                </List.Item>
                {/* 12 */}
                <List.Item c="blue">
                  <Text size="md" mb="md" c="black" fw={600}>
                    Applicability.
                  </Text>
                  <List type="ordered" listStyleType="lower-alpha">
                    {' '}
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        This Addendum will not apply to the Processing of Client Personal Data,
                        where such Processing is not regulated by EU Data Protection Laws. The
                        Parties to this Addendum hereby agree that the terms and conditions set out
                        herein shall be added as an addendum to the Service Agreement. Except where
                        the context requires otherwise, references in this Addendum to the Service
                        Agreement are to the Service Agreement as amended or supplemented by, and
                        including, this Addendum.
                      </Text>{' '}
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        The Terms of this Addendum shall take effect on the Effective Date and shall
                        continue concurrently for the term of the Service Agreement.
                      </Text>{' '}
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        If you are accepting the terms of this Addendum on behalf of an entity, you
                        represent and warrant to Cosmo Agents that you have the authority to bind
                        that entity and its affiliates, where applicable, to the terms and
                        conditions of this Addendum.
                      </Text>{' '}
                    </List.Item>
                  </List>
                </List.Item>
                {/* 13 */}
                <List.Item c="blue">
                  <Text size="md" mb="md" c="black" fw={600}>
                    Processing of Client Personal Data.
                  </Text>
                  <List type="ordered" listStyleType="lower-alpha">
                    {' '}
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        In the context of this Addendum, the Client acts as a Personal Data
                        Controller and Cosmo Agents acts as a Personal Data Processor with regard to
                        the Processing of Client Personal Data.
                      </Text>{' '}
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        Cosmo Agents shall:
                      </Text>{' '}
                      <List type="ordered" listStyleType="lower-roman">
                        <List.Item>
                          <Text size="md" mb="md" c="black">
                            comply with all EU Data Protection Laws in the Processing (as further
                            elaborated in{' '}
                            <Text span size="md" mb="md" td="underline" c="black" fw={600}>
                              Exhibit A
                            </Text>
                            , attached hereto and incorporated by reference) of Client Personal
                            Data;
                          </Text>{' '}
                        </List.Item>
                        <List.Item>
                          <Text size="md" mb="md" c="black">
                            not Process Client Personal Data other than on the Client’s relevant
                            documented instructions, including with regard to transfers of Client
                            Personal Data to a Third Country or an international organization,
                            unless such Processing is required by EU Data Protection Laws to which
                            the relevant Contracted Processor is subject, in which case Cosmo Agents
                            shall, to the extent permitted by EU Data Protection Laws, inform the
                            Client of that legal requirement before the applicable act of
                            Processing;
                          </Text>{' '}
                        </List.Item>
                        <List.Item>
                          <Text size="md" mb="md" c="black">
                            only conduct transfers of Client Personal Data, where such transfer
                            would otherwise be prohibited by EU Data Protection Laws due to there
                            being no applicable lawful exemption or derogation, in compliance with
                            all applicable conditions, as laid down in the EU Data Protection Laws;
                          </Text>{' '}
                        </List.Item>
                        <List.Item>
                          <Text size="md" mb="md" c="black">
                            not retain, delete, or otherwise Process Client Personal Data contrary
                            to or in the absence of the direct instructions of the Client, provided,
                            however, that the Client expressly and irrevocably authorizes such
                            retention, deletion or other Processing if and to the extent required or
                            allowed by any applicable law; and
                          </Text>{' '}
                        </List.Item>
                        <List.Item>
                          <Text size="md" mb="md" c="black">
                            immediately inform the Client in the event that, in Cosmo Agents’s
                            opinion, a Processing instruction given by the Client may infringe EU
                            Data Protection Laws.
                          </Text>{' '}
                        </List.Item>
                      </List>
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        The Client warrants that it will promptly update, when necessary, all
                        information provided during the process of acceptance of this Addendum,
                        including, where applicable, the contact details of its Data Protection
                        Officer and/or European Union Representative. Any such updates shall be sent
                        by email to{' '}
                        <Text size="md" mb="md" c="blue" td="underline" span>
                          dpo@Cosmo Agents.com.
                        </Text>{' '}
                      </Text>{' '}
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        The Client instructs Cosmo Agents (and authorizes Cosmo Agents to instruct
                        each Subprocessor) to Process Client Personal Data, and in particular,
                        transfer Client Personal Data to any country or territory, as reasonably
                        necessary for the provision of the Services and consistent with the Service
                        Agreement and this Addendum.
                      </Text>{' '}
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        The Client represents and warrants that it has all necessary rights to
                        provide the Client Personal Data to Cosmo Agents for the purpose of
                        Processing such data within the scope of this Addendum and the Service
                        Agreement.
                      </Text>{' '}
                    </List.Item>
                  </List>
                </List.Item>
                {/* 14 */}
                <List.Item c="blue">
                  <Text size="md" mb="md" c="black" fw={600}>
                    Cosmo Agents Personnel.
                  </Text>
                  <List type="ordered" listStyleType="lower-alpha">
                    {' '}
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        Cosmo Agents shall take reasonable steps to ensure the reliability of any
                        employee, agent, or contractor of any Contracted Processor who may have
                        access to the Client Personal Data, ensuring in each case that access is
                        strictly limited to those individuals who need to know or access the
                        relevant Client Personal Data, as strictly necessary for the purposes of the
                        Service Agreement, and to comply with EU Data Protection Laws in the context
                        of that individual’s duties to the Contracted Processor, ensuring that all
                        such individuals are subject to formal confidentiality undertakings or
                        professional or statutory obligations of confidentiality.
                      </Text>{' '}
                    </List.Item>
                  </List>
                </List.Item>
                {/* 15 */}
                <List.Item c="blue">
                  <Text size="md" mb="md" c="black" fw={600}>
                    Security of Processing.
                  </Text>
                  <List type="ordered" listStyleType="lower-alpha">
                    {' '}
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        Taking into account the state of the art, the costs of implementation and
                        the nature, scope, context, and purposes of Processing, as well as the risk
                        of varying likelihood and severity to the rights and freedoms of natural
                        persons, Cosmo Agents shall, with regard to Client Personal Data, implement
                        and maintain{' '}
                        <Text size="md" c="blue" span>
                          appropriate technical and organizational security measures
                        </Text>{' '}
                        to ensure a level of security appropriate to that risk, including, as
                        appropriate, the measures referred to in Article 32(1) of the GDPR as well
                        as assist the Client with regard to ensuring compliance with the obligations
                        pursuant to Article 32 of the GDPR borne directly by the Client.
                      </Text>{' '}
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        In assessing the appropriate level of security, Cosmo Agents shall take
                        account, in particular, of the risks that are presented by the nature of
                        such Processing activities, and particularly those related to possible
                        Personal Data Breaches.
                      </Text>{' '}
                    </List.Item>
                  </List>
                </List.Item>
                {/* 16 */}
                <List.Item c="blue">
                  <Text size="md" mb="md" c="black" fw={600}>
                    Subprocessing.
                  </Text>
                  <List type="ordered" listStyleType="lower-alpha">
                    {' '}
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        The Client authorizes Cosmo Agents to appoint (and permit each Subprocessor
                        appointed in accordance with this Section 6 to appoint) Subprocessors in
                        accordance with this Section 6 and any possible further restrictions, as set
                        out in the Service Agreement and this Addendum.
                      </Text>{' '}
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        Cosmo Agents may continue to use those Subprocessors already engaged by
                        Cosmo Agents as of the Effective Date subject to Cosmo Agents meeting the
                        obligations set out in Section 6.4. The list of Cosmo Agents’s
                        Subprocessors, current as of the Effective Date, is laid down in{' '}
                        <Text size="md" mb="md" c="black" span fw={600}>
                          Exhibit B
                        </Text>{' '}
                        , attached hereto and incorporated by reference.
                      </Text>{' '}
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        Cosmo Agents shall give the Client prior written notice of the appointment
                        of any new Subprocessor, by way of sending a notice. If, within 14 days of
                        sending of each such notice, the Client notifies Cosmo Agents in writing of
                        any reasonable objections to the proposed appointment, Cosmo Agents shall
                        not appoint or disclose any Client Personal Data to that proposed
                        Subprocessor until reasonable steps have been taken to address the
                        objections raised by the Client and, in turn, the Client has been provided
                        with a reasonable written explanation of the steps taken to account for any
                        such objections. If the Client, nevertheless, objects to the proposed
                        appointment, it shall be entitled to terminate the Service Agreement as a
                        remedy.
                      </Text>{' '}
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        With respect to each Subprocessor, Cosmo Agents shall:
                      </Text>{' '}
                      <List type="ordered" listStyleType="lower-roman">
                        <List.Item>
                          <Text size="md" mb="md" c="black">
                            carry out adequate due diligence to ensure that the Subprocessor is
                            capable of providing the level of protection for Client Personal Data
                            required by this Addendum, the Service Agreement, and EU Data Protection
                            Laws before the Subprocessor first Processes Client Personal Data or,
                            where applicable, in accordance with Section 6.2; and
                          </Text>{' '}
                        </List.Item>
                        <List.Item>
                          <Text size="md" mb="md" c="black">
                            ensure that the arrangement between: on the one hand, (i) Cosmo Agents,
                            or (ii) the relevant intermediate Subprocessor; and on the other hand,
                            the respective prospective Subprocessor, is governed by a written
                            contract, including terms which offer at least the same level of
                            protection for Client Personal Data as those set out in this Addendum,
                            and that such terms meet the requirements of Article 28(3) of the GDPR.
                          </Text>{' '}
                        </List.Item>
                      </List>
                    </List.Item>
                  </List>
                </List.Item>
                {/* 17 */}
                <List.Item c="blue">
                  <Text size="md" mb="md" c="black" fw={600}>
                    Rights of the Data Subjects.
                  </Text>
                  <List type="ordered" listStyleType="lower-alpha">
                    {' '}
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        Taking into account the nature of the Processing, Cosmo Agents shall assist
                        the Client by implementing appropriate technical and organizational
                        measures, insofar as this is possible, for the fulfilment of the Client’s
                        obligations, as reasonably understood by the Client, to respond to requests
                        to exercise Rights of the Data Subjects under the EU Data Protection Laws.
                      </Text>{' '}
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        With regard to Rights of the Data Subjects within the scope of this Section
                        7, Cosmo Agents shall:
                      </Text>{' '}
                      <List type="ordered" listStyleType="lower-roman">
                        <List.Item>
                          <Text size="md" mb="md" c="black">
                            promptly notify the Client if any Contracted Processor receives a
                            request from a Data Subject under any EU Data Protection Laws in respect
                            of Client Personal Data; and
                          </Text>{' '}
                        </List.Item>
                        <List.Item>
                          <Text size="md" mb="md" c="black">
                            ensure that the Contracted Processor does not respond to that request,
                            except on the documented instructions of the Client, or as required by
                            EU Data Protection Laws to which the Contracted Processor is subject, in
                            which case Cosmo Agents shall, to the extent permitted by EU Data
                            Protection Laws, inform the Client of that legal requirement before the
                            Contracted Processor responds to the request.
                          </Text>{' '}
                        </List.Item>
                      </List>
                    </List.Item>
                  </List>
                </List.Item>
                {/* 18 */}
                <List.Item c="blue">
                  <Text size="md" mb="md" c="black" fw={600}>
                    Restricted Transfers.
                  </Text>
                  <List type="ordered" listStyleType="lower-alpha">
                    {' '}
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        Cosmo Agents shall notify the Client without undue delay upon Cosmo Agents
                        becoming aware of a Personal Data Breach affecting Client Personal Data
                        under Salewhale’s direct control or upon Cosmo Agents being notified of a
                        Personal Data Breach affecting Client Personal Data under the direct control
                        of a Subprocessor, providing the Client with sufficient information to allow
                        the Client to meet any applicable obligations pursuant to the EU Data
                        Protection Laws, such as to report to the Supervisory Authorities or any
                        other competent authorities, or inform the Data Subjects of the Personal
                        Data Breach.
                      </Text>{' '}
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        Cosmo Agents shall co-operate with the Client and take all reasonable
                        commercial steps to assist the Client in the investigation, mitigation, and
                        remediation of each such Personal Data Breach.
                      </Text>{' '}
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        <Text size="md" mb="md" c="black" span>
                          Cosmo Agents shall co-operate with the Client and take all reasonable
                          commercial steps to assist the Client in the investigation, mitigation,
                          and remediation of each such Personal Data Breach.
                        </Text>{' '}
                      </Text>{' '}
                    </List.Item>
                  </List>
                </List.Item>
                {/* 19 */}
                <List.Item c="blue">
                  <Text size="md" mb="md" c="black" fw={600}>
                    Data Protection Impact Assessment and Prior Consultation.
                  </Text>
                  <List type="ordered" listStyleType="lower-alpha">
                    {' '}
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        Cosmo Agents shall provide the Client with relevant documentation, such as,
                        if available, an audit report (upon a written request and subject to
                        obligations of confidentiality), with regard to any data protection impact
                        assessments, and prior consultations with Supervisory Authorities or other
                        competent data privacy authorities, when the Client reasonably considers
                        that such data protection impact assessments or prior consultations are
                        required pursuant to Article 35 or 36 of the GDPR, or pursuant to the
                        equivalent provisions of any other EU Data Protection Laws but, in each such
                        case, solely with regard to Processing of Client Personal Data by, and
                        taking into account the nature of the Processing and information available
                        to, the respective Contracted Processors.
                      </Text>{' '}
                    </List.Item>
                  </List>
                </List.Item>
                {/* 20 */}
                <List.Item c="blue">
                  <Text size="md" mb="md" c="black" fw={600}>
                    Deletion or Return of Client Personal Data.
                  </Text>
                  <List type="ordered" listStyleType="lower-alpha">
                    {' '}
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        Cosmo Agents shall provide the Client with the technical means, consistent
                        with the way the Services are provided, to request the deletion of Client
                        Personal Data within the term of this Addendum and the Service Agreement,
                        unless EU Data Protection Laws require or allow storage of any such Client
                        Personal Data.
                      </Text>{' '}
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        Cosmo Agents shall promptly following the date of cessation of Services
                        involving the Processing of Client Personal Data, at the choice of the
                        Client, delete or return all Client Personal Data to the Client, as well as
                        delete existing copies, unless EU Data Protection Laws require or allow
                        storage of any such Client Personal Data.
                      </Text>{' '}
                    </List.Item>
                  </List>
                </List.Item>
                {/* 21 */}
                <List.Item c="blue">
                  <Text size="md" mb="md" c="black" fw={600}>
                    Audit Rights.
                  </Text>
                  <List type="ordered" listStyleType="lower-alpha">
                    {' '}
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        Where the Client is entitled to and desires to review Cosmo Agents’s
                        compliance with the EU Data Protection Laws, the Client may request, and
                        Cosmo Agents will provide (subject to obligations of confidentiality)
                        relevant documentation, or any relevant audit report Cosmo Agents might have
                        been issued. If the Client, after having reviewed such audit report(s),
                        still reasonably deems that it requires additional information, Cosmo Agents
                        shall further reasonably assist and make available to the Client, upon a
                        written request and subject to obligations of confidentiality, all other
                        information (excluding legal advice) and/or documentation necessary to
                        demonstrate compliance with this Addendum, and the obligations pursuant to
                        Articles 32 to 36 of the GDPR in particular, and shall allow for and
                        contribute to audits, including remote inspections of the Services, by the
                        Client or an auditor mandated by the Client with regard to the Processing of
                        the Client Personal Data by the Contracted Processors. Cosmo Agents shall
                        provide the assistance described in this Section 11, insofar as in Cosmo
                        Agents’s reasonable opinion such audits, and the specific requests of the
                        Client, do not interfere with Cosmo Agents’s business operations or cause
                        Cosmo Agents to breach any legal or contractual obligation to which it is
                        subject.
                      </Text>{' '}
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        The Client agrees to pay Cosmo Agents, upon receipt of invoice, a reasonable
                        fee based on the time spent, as well as to account for the materials
                        expended, in relation to the Client exercising its rights under this Section
                        11 or Clause 5(f) of the Standard Contractual Clauses, as set out in{' '}
                        <Text size="md" fw={600} td="underline" span>
                          Exhibit C
                        </Text>{' '}
                        , attached hereto and incorporated by reference, and which constitute an
                        integral part of this Addendum (the “
                        <Text size="md" fw={600} td="underline" span>
                          Standard Contractual Clauses
                        </Text>{' '}
                        ”).
                      </Text>{' '}
                    </List.Item>
                  </List>
                </List.Item>
                {/* 22 */}
                <List.Item c="blue">
                  <Text size="md" mb="md" c="black" fw={600}>
                    Restricted Transfers.
                  </Text>
                  <List type="ordered" listStyleType="lower-alpha">
                    {' '}
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        The Client (as “data exporter”) and Cosmo Agents (as “data importer”) hereby
                        enter into, as of the Effective Date, the Standard Contractual Clauses. The
                        Parties are deemed to have accepted and executed the Standard Contractual
                        Clauses in their entirety, including the appendices.
                      </Text>{' '}
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        With regard to any Restricted Transfer from the Client to Cosmo Agents
                        within the scope of this Addendum, one of the following transfer mechanisms
                        shall apply, in the following order of precedence:
                      </Text>{' '}
                      <List type="ordered" listStyleType="lower-roman">
                        <List.Item>
                          <Text size="md" mb="md" c="black">
                            Cosmo Agents’s EU-U.S. and Swiss-U.S. Privacy Shield Framework
                            self-certifications (if any and insofar as the prospective Restricted
                            Transfer would be considered lawful under this mechanism);
                          </Text>{' '}
                        </List.Item>
                        <List.Item>
                          <Text size="md" mb="md" c="black">
                            the Standard Contractual Clauses (insofar as the prospective Restricted
                            Transfer would be considered lawful under this mechanism); or
                          </Text>{' '}
                        </List.Item>
                        <List.Item>
                          <Text size="md" mb="md" c="black">
                            any other lawful basis, as laid down in EU Data Protection Laws.
                          </Text>{' '}
                        </List.Item>
                      </List>
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        In cases where the Standard Contractual Clauses apply and there is a
                        conflict between the terms of the Addendum and the terms of the Standard
                        Contractual Clauses, the terms of the Standard Contractual Clauses shall
                        control.
                      </Text>{' '}
                    </List.Item>
                  </List>
                </List.Item>
                {/* 23 */}
                <List.Item c="blue">
                  <Text size="md" mb="md" c="black" fw={600}>
                    General Terms.
                  </Text>
                  <List type="ordered" listStyleType="lower-alpha">
                    {' '}
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        All clauses of the Service Agreement that are not explicitly amended or
                        supplemented by the clauses of this Addendum shall remain in full force and
                        effect and shall apply so long as they do not contradict Applicable Laws.
                      </Text>{' '}
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        Cosmo Agents may amend the terms of this Addendum, insofar as the revised
                        Addendum continues to comply with the relevant requirements of the EU Data
                        Protection Laws, upon notice to the Client by email to the primary contact
                        on the account. Any such amendments will automatically become effective 10
                        days after Cosmo Agents’s transmission of each such notice.
                      </Text>{' '}
                    </List.Item>
                    <List.Item>
                      <Text size="md" mb="md" c="black">
                        In the event of any conflict between the Service Agreement (including any
                        annexes and appendices thereto) and this Addendum, the provisions of this
                        Addendum shall control.
                      </Text>{' '}
                    </List.Item>
                  </List>
                </List.Item>
              </List>
              <Text size="md" mb="md">
                Should any provision of this Addendum be found invalid or unenforceable pursuant to
                any applicable law, then the invalid or unenforceable provision will be deemed
                superseded by a valid, enforceable provision that most closely matches the intent of
                the original provision and the remainder of the Addendum will continue in effect.
              </Text>
              <Text size="md" mb="md">
                If Cosmo Agents makes a determination that it can no longer meet its obligations in
                accordance with this Addendum, it shall promptly notify the Client of that
                determination, and cease the Processing or take other reasonable and appropriate
                steps to remediate.
              </Text>
              {/* Data Protection Officer */}
              {/* <Box>
                <Text size="md" mb="md" fw={600}>
                  Data Protection Officer.
                </Text>
                <Text size="md" mb="md">
                  Cosmo Agents appointed{' '}
                  <Text size="md" mb="md" c="blue" td="underline" span>
                    VeraSafe
                  </Text>{' '}
                  as its Data Protection Officer (DPO): VeraSafe
                </Text>
                <Text size="md" mb="md">
                  22 Essex Way #8203
                </Text>
                <Text size="md" mb="md">
                  Essex, VT 05451 USA
                </Text>
                <Text size="md" mb="md">
                  Phone: +1 (617) 398-7069
                </Text>
                <Text size="md" mb="md">
                  Email:{' '}
                  <Text size="md" mb="md" c="blue" td="underline" span>
                    experts@verasafe.com
                  </Text>
                </Text>
              </Box> */}

              <Title order={2} mt="lg" mb="md" id="exhibit-a" style={{ scrollMarginTop: '80px' }}>
                Exhibit A
              </Title>
              {/* Exhibit A Content */}
              <Box>
                <List type="ordered" c="blue">
                  <List.Item>
                    <Text size="md" mb="md" c="black">
                      Pursuant to Article 28(3) of the GDPR, further details of the Processing, in
                      addition to the ones laid down in the Service Agreement and this Addendum,
                      include:
                    </Text>
                    <Text size="md" mb="md" c="black">
                      The subject matter of the Processing of Client Personal Data is:
                    </Text>
                    <Text size="md" mb="md" c="black">
                      The subject matter of the Processing of Client Personal Data pertains to the
                      provision of Services, as requested by the Client.
                    </Text>
                  </List.Item>
                  <List.Item>
                    <Text size="md" mb="md" c="black">
                      The duration of the Processing of Client Personal Data is:
                    </Text>
                    <Text size="md" mb="md" c="black">
                      The duration of the Processing of Client Personal Data is generally determined
                      by the Client and is further subject to the term of this Addendum and the
                      Service Agreement, respectively, in the context of the contractual
                      relationship between Cosmo Agents and the Client.
                    </Text>
                  </List.Item>
                  <List.Item>
                    <Text size="md" mb="md" c="black">
                      The nature and purpose of the Processing of Client Personal Data is:
                    </Text>
                    <Text size="md" mb="md" c="black">
                      The purpose of Processing of Client Personal Data pertains to the provision of
                      sales assistance, as requested by the Client. The nature of such Processing is
                      related to these purposes and is elaborated on in this Addendum and the
                      Service Agreement.
                    </Text>
                  </List.Item>
                  <List.Item>
                    <Text size="md" mb="md" c="black">
                      The types of Client Personal Data to be Processed include:
                    </Text>
                    <Text size="md" mb="md" c="black">
                      Biographical data, such as name, email address, phone number; meta data; any
                      other category of Personal Data that could be included in an email.
                    </Text>
                  </List.Item>
                  <List.Item>
                    <Text size="md" mb="md" c="black">
                      The categories of Data Subjects to whom the Client Personal Data relates
                      include:
                    </Text>
                    <Text size="md" mb="md" c="black">
                      Sales prospects of the Client (Cosmo Agents’s customer).
                    </Text>
                  </List.Item>
                  <List.Item>
                    <Text size="md" mb="md" c="black">
                      The obligations and rights of the Client are:
                    </Text>
                    <Text size="md" mb="md" c="black">
                      The rights and obligations of the Client are set out in the Service Agreement
                      and this Addendum.
                    </Text>
                  </List.Item>
                </List>
              </Box>
              <Title order={2} mt="lg" mb="md" id="exhibit-b" style={{ scrollMarginTop: '80px' }}>
                Exhibit B
              </Title>
              {/* Exhibit B Content */}
              <Box>
                <Text size="md" mb="md" ta="center">
                  List of Subprocessors
                </Text>
                <Text size="md" mb="md">
                  Below is a list of the Subprocessors of Cosmo Agents, current as of the Effective
                  Date, pursuant to Article 6.2 of the Addendum:
                </Text>
                <List type="ordered" c="blue">
                  <List.Item>
                    <Text size="md" mb="md" c="black">
                      AWS, Inc. (Amazon Web Services) – U.S.A.
                    </Text>
                  </List.Item>
                  <List.Item>
                    <Text size="md" mb="md" c="black">
                      Google LLC (Google Drive) – U.S.A.
                    </Text>
                  </List.Item>
                  <List.Item>
                    <Text size="md" mb="md" c="black">
                      Rockship Pte. Ltd. ‒ Singapore.
                    </Text>
                  </List.Item>
                </List>
              </Box>

              <Title order={2} mt="lg" mb="md" id="exhibit-c" style={{ scrollMarginTop: '80px' }}>
                Exhibit C
              </Title>
              {/* Exhibit C Content  */}
              <Box>
                <Text size="md" ta="center" fw={600}>
                  Commission Decision C(2010)593
                </Text>
                <Text size="md" mb="md" ta="center" fw={600}>
                  Standard Contractual Clauses (processors)
                </Text>
                <Text size="md" mb="md">
                  For the purposes of Article 26(2) of Directive 95/46/EC for the transfer of
                  personal data to processors established in third countries which do not ensure an
                  adequate level of data protection,
                </Text>
                <Text size="md" mb="md">
                  the Client, as defined in the Addendum (as “data exporter”),
                </Text>
                <Text size="md" mb="md">
                  and Cosmo Agents, as defined in the Addendum (as “data importer”) each a “party”;
                  together “the parties”,
                </Text>
                <Text size="md" mb="md">
                  HAVE AGREED on the following Contractual Clauses (the Clauses) in order to adduce
                  adequate safeguards with respect to the protection of privacy and fundamental
                  rights and freedoms of individuals for the transfer by the data exporter to the
                  data importer of the personal data specified in Appendix 1.
                </Text>
                {/* Clause 1 */}
                <Box>
                  <Text size="md" mb="md" fs="italic" ta="center">
                    Clause 1
                  </Text>
                  <Text size="md" mb="md" fs="italic" ta="center" fw={600}>
                    Definitions
                  </Text>
                  <Text size="md" mb="md">
                    For the purposes of the Clauses:
                  </Text>
                  <Text size="md" mb="md">
                    (a)
                    <Text size="md" mb="md" fs="italic" span>
                      ‘personal data’, ‘special categories of data’, ‘process/processing’,
                      ‘controller’, ‘processor’, ‘data subject’ and ‘supervisory authority’
                    </Text>
                    shall have the same meaning as in Directive 95/46/EC of the European Parliament
                    and of the Council of 24 October 1995 on the protection of individuals with
                    regard to the processing of personal data and on the free movement of such data;
                  </Text>
                  <Text size="md" mb="md">
                    (b)
                    <Text size="md" mb="md" fs="italic" span>
                      ‘the data exporter’{' '}
                    </Text>
                    means the controller who transfers the personal data;
                  </Text>
                  <Text size="md" mb="md">
                    (c)
                    <Text size="md" mb="md" fs="italic" span>
                      ‘the data importer’{' '}
                    </Text>
                    means the processor who agrees to receive from the data exporter personal data
                    intended for processing on his behalf after the transfer in accordance with his
                    instructions and the terms of the Clauses and who is not subject to a third
                    country’s system ensuring adequate protection within the meaning of Article
                    25(1) of Directive 95/46/EC;
                  </Text>
                  <Text size="md" mb="md">
                    (d)
                    <Text size="md" mb="md" fs="italic" span>
                      ‘the subprocessor’{' '}
                    </Text>
                    means any processor engaged by the data importer or by any other subprocessor of
                    the data importer who agrees to receive from the data importer or from any other
                    subprocessor of the data importer personal data exclusively intended for
                    processing activities to be carried out on behalf of the data exporter after the
                    transfer in accordance with his instructions, the terms of the Clauses and the
                    terms of the written subcontract;
                  </Text>
                  <Text size="md" mb="md">
                    (e)
                    <Text size="md" mb="md" fs="italic" span>
                      ‘the applicable data protection law’{' '}
                    </Text>
                    means the legislation protecting the fundamental rights and freedoms of
                    individuals and, in particular, their right to privacy with respect to the
                    processing of personal data applicable to a data controller in the Member State
                    in which the data exporter is established;
                  </Text>
                  <Text size="md" mb="md">
                    (f)
                    <Text size="md" mb="md" fs="italic" span>
                      ‘technical and organisational security measures’{' '}
                    </Text>
                    means those measures aimed at protecting personal data against accidental or
                    unlawful destruction or accidental loss, alteration, unauthorised disclosure or
                    access, in particular where the processing involves the transmission of data
                    over a network, and against all other unlawful forms of processing.
                  </Text>
                </Box>
                {/* Clause 2 */}
                <Box>
                  <Text size="md" mb="md" fs="italic" ta="center">
                    Clause 2
                  </Text>
                  <Text size="md" mb="md" fs="italic" ta="center" fw={600}>
                    Details of the transfer
                  </Text>
                  <Text size="md" mb="md">
                    The details of the transfer and in particular the special categories of personal
                    data where applicable are specified in Appendix 1 which forms an integral part
                    of the Clauses.
                  </Text>
                </Box>
                {/* Clause 3 */}
                <Box>
                  <Text size="md" mb="md" fs="italic" ta="center">
                    Clause 3
                  </Text>
                  <Text size="md" mb="md" fs="italic" ta="center" fw={600}>
                    Third-party beneficiary clause
                  </Text>
                  <Text size="md" mb="md">
                    (a) The data subject can enforce against the data exporter this Clause, Clause
                    4(b) to (i), Clause 5(a) to (e), and (g) to (j), Clause 6(1) and (2), Clause 7,
                    Clause 8(2), and Clauses 9 to 12 as third-party beneficiary.
                  </Text>
                  <Text size="md" mb="md">
                    (b) The data subject can enforce against the data importer this Clause, Clause
                    5(a) to (e) and (g), Clause 6, Clause 7, Clause 8(2), and Clauses 9 to 12, in
                    cases where the data exporter has factually disappeared or has ceased to exist
                    in law unless any successor entity has assumed the entire legal obligations of
                    the data exporter by contract or by operation of law, as a result of which it
                    takes on the rights and obligations of the data exporter, in which case the data
                    subject can enforce them against such entity.
                  </Text>
                  <Text size="md" mb="md">
                    (c) The data subject can enforce against the subprocessor this Clause, Clause
                    5(a) to (e) and (g), Clause 6, Clause 7, Clause 8(2), and Clauses 9 to 12, in
                    cases where both the data exporter and the data importer have factually
                    disappeared or ceased to exist in law or have become insolvent, unless any
                    successor entity has assumed the entire legal obligations of the data exporter
                    by contract or by operation of law as a result of which it takes on the rights
                    and obligations of the data exporter, in which case the data subject can enforce
                    them against such entity. Such third-party liability of the subprocessor shall
                    be limited to its own processing operations under the Clauses.
                  </Text>
                  <Text size="md" mb="md">
                    (d) The parties do not object to a data subject being represented by an
                    association or other body if the data subject so expressly wishes and if
                    permitted by national law.
                  </Text>
                </Box>
                {/* Clause 4 */}
                <Box>
                  <Text size="md" mb="md" fs="italic" ta="center">
                    Clause 4
                  </Text>
                  <Text size="md" mb="md" fs="italic" ta="center" fw={600}>
                    Obligations of the data exporter
                  </Text>
                  <Text size="md" mb="md">
                    The data exporter agrees and warrants:
                  </Text>
                  <Text size="md" mb="md">
                    (a) that the processing, including the transfer itself, of the personal data has
                    been and will continue to be carried out in accordance with the relevant
                    provisions of the applicable data protection law (and, where applicable, has
                    been notified to the relevant authorities of the Member State where the data
                    exporter is established) and does not violate the relevant provisions of that
                    State;
                  </Text>
                  <Text size="md" mb="md">
                    (b) that it has instructed and throughout the duration of the personal data
                    processing services will instruct the data importer to process the personal data
                    transferred only on the data exporter’s behalf and in accordance with the
                    applicable data protection law and the Clauses;
                  </Text>
                  <Text size="md" mb="md">
                    (c) that the data importer will provide sufficient guarantees in respect of the
                    technical and organisational security measures specified in Appendix 2 to this
                    contract;
                  </Text>
                  <Text size="md" mb="md">
                    (d) that after assessment of the requirements of the applicable data protection
                    law, the security measures are appropriate to protect personal data against
                    accidental or unlawful destruction or accidental loss, alteration, unauthorised
                    disclosure or access, in particular where the processing involves the
                    transmission of data over a network, and against all other unlawful forms of
                    processing, and that these measures ensure a level of security appropriate to
                    the risks presented by the processing and the nature of the data to be protected
                    having regard to the state of the art and the cost of their implementation;
                  </Text>
                  <Text size="md" mb="md">
                    (e) that it will ensure compliance with the security measures;
                  </Text>
                  <Text size="md" mb="md">
                    (f) that, if the transfer involves special categories of data, the data subject
                    has been informed or will be informed before, or as soon as possible after, the
                    transfer that its data could be transmitted to a third country not providing
                    adequate protection within the meaning of Directive 95/46/EC;
                  </Text>
                  <Text size="md" mb="md">
                    (g) to forward any notification received from the data importer or any
                    subprocessor pursuant to Clause 5(b) and Clause 8(3) to the data protection
                    supervisory authority if the data exporter decides to continue the transfer or
                    to lift the suspension;
                  </Text>
                  <Text size="md" mb="md">
                    (h) to make available to the data subjects upon request a copy of the Clauses,
                    with the exception of Appendix 2, and a summary description of the security
                    measures, as well as a copy of any contract for subprocessing services which has
                    to be made in accordance with the Clauses, unless the Clauses or the contract
                    contain commercial information, in which case it may remove such commercial
                    information;
                  </Text>
                  <Text size="md" mb="md">
                    (i) that, in the event of subprocessing, the processing activity is carried out
                    in accordance with Clause 11 by a subprocessor providing at least the same level
                    of protection for the personal data and the rights of data subject as the data
                    importer under the Clauses; and
                  </Text>
                  <Text size="md" mb="md">
                    (j) that it will ensure compliance with Clause 4(a) to (i).
                  </Text>
                </Box>
                {/* Clause 5 */}
                <Box>
                  <Text size="md" mb="md" fs="italic" ta="center">
                    Clause 5
                  </Text>
                  <Text size="md" mb="md" fs="italic" ta="center" fw={600}>
                    Obligations of the data importer
                  </Text>
                  <Text size="md" mb="md">
                    The data importer agrees and warrants:
                  </Text>
                  <Text size="md" mb="md">
                    (a) to process the personal data only on behalf of the data exporter and in
                    compliance with its instructions and the Clauses; if it cannot provide such
                    compliance for whatever reasons, it agrees to inform promptly the data exporter
                    of its inability to comply, in which case the data exporter is entitled to
                    suspend the transfer of data and/or terminate the contract;
                  </Text>
                  <Text size="md" mb="md">
                    (b) that it has no reason to believe that the legislation applicable to it
                    prevents it from fulfilling the instructions received from the data exporter and
                    its obligations under the contract and that in the event of a change in this
                    legislation which is likely to have a substantial adverse effect on the
                    warranties and obligations provided by the Clauses, it will promptly notify the
                    change to the data exporter as soon as it is aware, in which case the data
                    exporter is entitled to suspend the transfer of data and/or terminate the
                    contract;
                  </Text>
                  <Text size="md" mb="md">
                    (c) that it has implemented the technical and organisational security measures
                    specified in Appendix 2 before processing the personal data transferred;
                  </Text>
                  <Text size="md" mb="md">
                    (d) that it will promptly notify the data exporter about:
                  </Text>
                  <Text size="md" ml="xl" my="md">
                    (1) any legally binding request for disclosure of the personal data by a law
                    enforcement authority unless otherwise prohibited, such as a prohibition under
                    criminal law to preserve the confidentiality of a law enforcement investigation,
                  </Text>
                  <Text size="md" ml="xl" my="md">
                    (2) any accidental or unauthorised access, and
                  </Text>
                  <Text size="md" ml="xl" my="md">
                    (3) any request received directly from the data subjects without responding to
                    that request, unless it has been otherwise authorised to do so;
                  </Text>
                  <Text size="md" mb="md">
                    (e) to deal promptly and properly with all inquiries from the data exporter
                    relating to its processing of the personal data subject to the transfer and to
                    abide by the advice of the supervisory authority with regard to the processing
                    of the data transferred;
                  </Text>
                  <Text size="md" mb="md">
                    (f) at the request of the data exporter to submit its data processing facilities
                    for audit of the processing activities covered by the Clauses which shall be
                    carried out by the data exporter or an inspection body composed of independent
                    members and in possession of the required professional qualifications bound by a
                    duty of confidentiality, selected by the data exporter, where applicable, in
                    agreement with the supervisory authority;
                  </Text>
                  <Text size="md" mb="md">
                    (g) to make available to the data subject upon request a copy of the Clauses, or
                    any existing contract for subprocessing, unless the Clauses or contract contain
                    commercial information, in which case it may remove such commercial information,
                    with the exception of Appendix 2 which shall be replaced by a summary
                    description of the security measures in those cases where the data subject is
                    unable to obtain a copy from the data exporter;
                  </Text>
                  <Text size="md" mb="md">
                    (h) that, in the event of subprocessing, it has previously informed the data
                    exporter and obtained its prior written consent;
                  </Text>
                  <Text size="md" mb="md">
                    (i) that the processing services by the subprocessor will be carried out in
                    accordance with Clause 11;
                  </Text>
                  <Text size="md" mb="md">
                    (j) to send promptly a copy of any subprocessor agreement it concludes under the
                    Clauses to the data exporter.
                  </Text>
                </Box>
                {/* Clause 6 */}
                <Box>
                  <Text size="md" mb="md" fs="italic" ta="center">
                    Clause 6
                  </Text>
                  <Text size="md" mb="md" fs="italic" ta="center" fw={600}>
                    Liability
                  </Text>
                  <Text size="md" mb="md">
                    (a) The parties agree that any data subject, who has suffered damage as a result
                    of any breach of the obligations referred to in Clause 3 or in Clause 11 by any
                    party or subprocessor is entitled to receive compensation from the data exporter
                    for the damage suffered.
                  </Text>
                  <Text size="md" mb="md">
                    (b) If a data subject is not able to bring a claim for compensation in
                    accordance with paragraph 1 against the data exporter, arising out of a breach
                    by the data importer or his subprocessor of any of their obligations referred to
                    in Clause 3 or in Clause 11, because the data exporter has factually disappeared
                    or ceased to exist in law or has become insolvent, the data importer agrees that
                    the data subject may issue a claim against the data importer as if it were the
                    data exporter, unless any successor entity has assumed the entire legal
                    obligations of the data exporter by contract of by operation of law, in which
                    case the data subject can enforce its rights against such entity.
                  </Text>
                  <Text size="md" mb="md">
                    The data importer may not rely on a breach by a subprocessor of its obligations
                    in order to avoid its own liabilities.
                  </Text>
                  <Text size="md" mb="md">
                    (c) If a data subject is not able to bring a claim against the data exporter or
                    the data importer referred to in paragraphs 1 and 2, arising out of a breach by
                    the subprocessor of any of their obligations referred to in Clause 3 or in
                    Clause 11 because both the data exporter and the data importer have factually
                    disappeared or ceased to exist in law or have become insolvent, the subprocessor
                    agrees that the data subject may issue a claim against the data subprocessor
                    with regard to its own processing operations under the Clauses as if it were the
                    data exporter or the data importer, unless any successor entity has assumed the
                    entire legal obligations of the data exporter or data importer by contract or by
                    operation of law, in which case the data subject can enforce its rights against
                    such entity. The liability of the subprocessor shall be limited to its own
                    processing operations under the Clauses.
                  </Text>
                </Box>
                {/* Clause 7 */}
                <Box>
                  <Text size="md" mb="md" fs="italic" ta="center">
                    Clause 7
                  </Text>
                  <Text size="md" mb="md" fs="italic" ta="center" fw={600}>
                    Mediation and jurisdiction
                  </Text>
                  <Text size="md" mb="md">
                    (a) The data importer agrees that if the data subject invokes against it
                    third-party beneficiary rights and/or claims compensation for damages under the
                    Clauses, the data importer will accept the decision of the data subject:
                  </Text>
                  <Text size="md" mb="md">
                    (b) to refer the dispute to mediation, by an independent person or, where
                    applicable, by the supervisory authority;
                  </Text>
                  <Text size="md" mb="md">
                    (c) to refer the dispute to the courts in the Member State in which the data
                    exporter is established.
                  </Text>
                  <Text size="md" mb="md">
                    (d) The parties agree that the choice made by the data subject will not
                    prejudice its substantive or procedural rights to seek remedies in accordance
                    with other provisions of national or international law.
                  </Text>
                </Box>
                {/* Clause 8 */}
                <Box>
                  <Text size="md" mb="md" fs="italic" ta="center">
                    Clause 8
                  </Text>
                  <Text size="md" mb="md" fs="italic" ta="center" fw={600}>
                    Cooperation with supervisory authorities
                  </Text>
                  <Text size="md" mb="md">
                    (a) The data exporter agrees to deposit a copy of this contract with the
                    supervisory authority if it so requests or if such deposit is required under the
                    applicable data protection law.
                  </Text>
                  <Text size="md" mb="md">
                    (b) The parties agree that the supervisory authority has the right to conduct an
                    audit of the data importer, and of any subprocessor, which has the same scope
                    and is subject to the same conditions as would apply to an audit of the data
                    exporter under the applicable data protection law.
                  </Text>
                  <Text size="md" mb="md">
                    (c) The data importer shall promptly inform the data exporter about the
                    existence of legislation applicable to it or any subprocessor preventing the
                    conduct of an audit of the data importer, or any subprocessor, pursuant to
                    paragraph 2. In such a case the data exporter shall be entitled to take the
                    measures foreseen in Clause 5 (b).
                  </Text>
                </Box>
                {/* Clause 9 */}
                <Box>
                  <Text size="md" mb="md" fs="italic" ta="center">
                    Clause 9
                  </Text>
                  <Text size="md" mb="md" fs="italic" ta="center" fw={600}>
                    Governing Law
                  </Text>
                  <Text size="md" mb="md">
                    The Clauses shall be governed by the law of the Member State in which the data
                    exporter is established.
                  </Text>
                </Box>
                {/* Clause 10 */}
                <Box>
                  <Text size="md" mb="md" fs="italic" ta="center">
                    Clause 10
                  </Text>
                  <Text size="md" mb="md" fs="italic" ta="center" fw={600}>
                    Variation of the contract
                  </Text>
                  <Text size="md" mb="md">
                    The parties undertake not to vary or modify the Clauses. This does not preclude
                    the parties from adding clauses on business related issues where required as
                    long as they do not contradict the Clause.
                  </Text>
                </Box>
                {/* Clause 11 */}
                <Box>
                  <Text size="md" mb="md" fs="italic" ta="center">
                    Clause 11
                  </Text>
                  <Text size="md" mb="md" fs="italic" ta="center" fw={600}>
                    Subprocessing
                  </Text>
                  <Text size="md" mb="md">
                    (a) The data importer shall not subcontract any of its processing operations
                    performed on behalf of the data exporter under the Clauses without the prior
                    written consent of the data exporter. Where the data importer subcontracts its
                    obligations under the Clauses, with the consent of the data exporter, it shall
                    do so only by way of a written agreement with the subprocessor which imposes the
                    same obligations on the subprocessor as are imposed on the data importer under
                    the Clauses. Where the subprocessor fails to fulfil its data protection
                    obligations under such written agreement the data importer shall remain fully
                    liable to the data exporter for the performance of the subprocessor’s
                    obligations under such agreement.
                  </Text>
                  <Text size="md" mb="md">
                    (b) The prior written contract between the data importer and the subprocessor
                    shall also provide for a third-party beneficiary clause as laid down in Clause 3
                    for cases where the data subject is not able to bring the claim for compensation
                    referred to in paragraph 1 of Clause 6 against the data exporter or the data
                    importer because they have factually disappeared or have ceased to exist in law
                    or have become insolvent and no successor entity has assumed the entire legal
                    obligations of the data exporter or data importer by contract or by operation of
                    law. Such third-party liability of the subprocessor shall be limited to its own
                    processing operations under the Clauses.
                  </Text>
                  <Text size="md" mb="md">
                    (c) The provisions relating to data protection aspects for subprocessing of the
                    contract referred to in paragraph 1 shall be governed by the law of the Member
                    State in which the data exporter is established.
                  </Text>
                  <Text size="md" mb="md">
                    (d) The data exporter shall keep a list of subprocessing agreements concluded
                    under the Clauses and notified by the data importer pursuant to Clause 5 (j),
                    which shall be updated at least once a year. The list shall be available to the
                    data exporter’s data protection supervisory authority.
                  </Text>
                </Box>
                {/* Clause 12 */}
                <Box>
                  <Text size="md" mb="md" fs="italic" ta="center">
                    Clause 12
                  </Text>
                  <Text size="md" mb="md" fs="italic" ta="center" fw={600}>
                    Obligation after the termination of personal data processing services
                  </Text>
                  <Text size="md" mb="md">
                    (a) The parties agree that on the termination of the provision of data
                    processing services, the data importer and the subprocessor shall, at the choice
                    of the data exporter, return all the personal data transferred and the copies
                    thereof to the data exporter or shall destroy all the personal data and certify
                    to the data exporter that it has done so, unless legislation imposed upon the
                    data importer prevents it from returning or destroying all or part of the
                    personal data transferred. In that case, the data importer warrants that it will
                    guarantee the confidentiality of the personal data transferred and will not
                    actively process the personal data transferred anymore.
                  </Text>
                  <Text size="md" mb="md">
                    (b) The data importer and the subprocessor warrant that upon request of the data
                    exporter and/or of the supervisory authority, it will submit its data processing
                    facilities for an audit of the measures referred to in paragraph 1.
                  </Text>
                  <Text size="md" mb="md" td="underline" ta="center" fw={600}>
                    Appendix 1 to the Standard Contractual Clauses
                  </Text>
                  <Text size="md" mb="md" td="underline" fw={600}>
                    By entering into the Standard Contractual Clauses, pursuant to Section 12.1 of
                    the Addendum, the parties are deemed to have signed this Appendix 1.
                  </Text>
                  <Text size="md" mb="md" fw={600}>
                    Data exporter
                  </Text>
                  <Text size="md" mb="md">
                    The data exporter is the Client, as defined in the Addendum
                  </Text>
                  <Text size="md" mb="md" fw={600}>
                    Data importer
                  </Text>
                  <Text size="md" mb="md">
                    The data importer is Cosmo Agents, as defined in the Addendum
                  </Text>
                  <Text size="md" mb="md" fw={600}>
                    Data subjects
                  </Text>
                  <Text size="md" mb="md">
                    As indicated under Section 1.5 of Exhibit A of the Addendum.
                  </Text>
                  <Text size="md" mb="md" fw={600}>
                    Categories of data
                  </Text>
                  <Text size="md" mb="md">
                    As indicated under Section 1.4 of Exhibit A of the Addendum.
                  </Text>
                  <Text size="md" mb="md" fw={600}>
                    Processing operations
                  </Text>
                  <Text size="md" mb="md">
                    As indicated under Section 1.3 of Exhibit A of the Addendum.
                  </Text>
                  <Text size="md" mb="md" td="underline" ta="center" fw={600}>
                    Appendix 2 to the Standard Contractual Clauses
                  </Text>
                  <Text size="md" mb="md" td="underline" fw={600}>
                    By entering into the Standard Contractual Clauses, pursuant to Section 12.1 of
                    the Addendum, the parties are deemed to have signed this Appendix 2.
                  </Text>
                  <Text size="md" mb="md" fw={600}>
                    Description of the technical and organisational security measures implemented by
                    the data importer in accordance with Clauses 4(d) and 5(c) (or
                    document/legislation attached):
                  </Text>
                  <Text size="md" mb="md">
                    Data importer has implemented and will maintain the technical and organizational
                    security measures to ensure a level of security appropriate to the risk,
                    including, as appropriate, the measures referred to in Article 32(1) of the
                    GDPR.
                  </Text>
                </Box>
              </Box>
            </Box>
          </AppShell.Main>
        </AppShell>
      </Box>
    </Box>
  );
}
