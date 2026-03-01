DO $$
DECLARE
    v_user_id uuid;
    v_org_id uuid;
    v_segment_id uuid := gen_random_uuid();
    v_contact_ids uuid[];
BEGIN
    SELECT id INTO v_user_id
    FROM users
    WHERE is_deleted = false
    ORDER BY created_at DESC
    LIMIT 1;

    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'No users found. Create a user first.';
    END IF;

    SELECT id INTO v_org_id
    FROM organizations
    WHERE user_id = v_user_id AND is_deleted = false
    ORDER BY created_at DESC
    LIMIT 1;

    INSERT INTO segmentations (
        id, user_id, name, description, priority, criteria, icp_definition, is_active
    ) VALUES (
        v_segment_id,
        v_user_id,
        'Playbook Test Segment',
        'Seeded segment for playbook automation tests',
        5,
        '{"filters": [], "scoring_rules": [], "exclusions": []}'::jsonb,
        '{}'::jsonb,
        true
    )
    ON CONFLICT (id) DO NOTHING;

    WITH inserted AS (
        INSERT INTO contacts (
            id, user_id, organization_id, source_id, source,
            first_name, last_name, email, phone,
            company, job_title, address, city, country, state, zip,
            profile, confirmed_facts, ai_insights, insight_validation, scores,
            do_not_contact, tags
        ) VALUES
        (
            gen_random_uuid(), v_user_id, v_org_id, gen_random_uuid()::text, 'cosmo-agents',
            'Linh', 'Tran', 'linh.playbook1@example.com', 'N/A',
            'Acme', 'Sales Lead', 'N/A', 'Ho Chi Minh City', 'Vietnam', 'N/A', 'N/A',
            '{}'::jsonb, '{}'::jsonb, '{}'::jsonb, '{}'::jsonb, '{"fit": 90, "engagement": 60, "priority": 80}'::jsonb,
            false, '{}'::jsonb
        ),
        (
            gen_random_uuid(), v_user_id, v_org_id, gen_random_uuid()::text, 'cosmo-agents',
            'Minh', 'Pham', 'minh.playbook2@example.com', 'N/A',
            'Globex', 'VP Sales', 'N/A', 'Ha Noi', 'Vietnam', 'N/A', 'N/A',
            '{}'::jsonb, '{}'::jsonb, '{}'::jsonb, '{}'::jsonb, '{"fit": 88, "engagement": 55, "priority": 75}'::jsonb,
            false, '{}'::jsonb
        ),
        (
            gen_random_uuid(), v_user_id, v_org_id, gen_random_uuid()::text, 'cosmo-agents',
            'Mai', 'Nguyen', 'mai.playbook3@example.com', 'N/A',
            'Initech', 'Head of Growth', 'N/A', 'Da Nang', 'Vietnam', 'N/A', 'N/A',
            '{}'::jsonb, '{}'::jsonb, '{}'::jsonb, '{}'::jsonb, '{"fit": 92, "engagement": 70, "priority": 85}'::jsonb,
            false, '{}'::jsonb
        )
        RETURNING id
    )
    SELECT array_agg(id) INTO v_contact_ids FROM inserted;

    INSERT INTO contact_segment_scores (
        id, contact_id, segmentation_id, fit_score, score_breakdown, score_type, status, passes_filters, enrolled_in_campaign
    )
    SELECT
        gen_random_uuid(), contact_id, v_segment_id, 90, '{}'::jsonb, 'manual', 'qualified', true, false
    FROM unnest(v_contact_ids) AS contact_id;

    RAISE NOTICE 'Seeded user_id=%', v_user_id;
    RAISE NOTICE 'Seeded organization_id=%', v_org_id;
    RAISE NOTICE 'Seeded segmentation_id=%', v_segment_id;
    RAISE NOTICE 'Seeded contact_ids=%', v_contact_ids;
END $$;
