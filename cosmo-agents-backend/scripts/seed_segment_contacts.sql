DO $$
DECLARE
    v_user_id uuid;
    v_org_id uuid;
    v_segment_id uuid := gen_random_uuid();
    v_contact_ids uuid[];
BEGIN
    v_user_id := 'a8ef50a7-da8e-44df-8002-c0a9353e0746';

    IF v_user_id IS NULL THEN
        RAISE EXCEPTION 'User ID not set.';
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
        'Priority Meetings',
        'Seeded segment for UI filter testing',
        9,
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
            'Tuan', 'Nguyen', 'tuan.meeting1@example.com', 'N/A',
            'Orbit', 'Account Executive', 'N/A', 'Ho Chi Minh City', 'Vietnam', 'N/A', 'N/A',
            '{}'::jsonb, '{}'::jsonb, '{}'::jsonb, '{}'::jsonb, '{"fit": 96, "engagement": 80, "priority": 90}'::jsonb,
            false, '{}'::jsonb
        ),
        (
            gen_random_uuid(), v_user_id, v_org_id, gen_random_uuid()::text, 'cosmo-agents',
            'Thao', 'Le', 'thao.meeting2@example.com', 'N/A',
            'Nimbus', 'Sales Manager', 'N/A', 'Ha Noi', 'Vietnam', 'N/A', 'N/A',
            '{}'::jsonb, '{}'::jsonb, '{}'::jsonb, '{}'::jsonb, '{"fit": 91, "engagement": 75, "priority": 85}'::jsonb,
            false, '{}'::jsonb
        ),
        (
            gen_random_uuid(), v_user_id, v_org_id, gen_random_uuid()::text, 'cosmo-agents',
            'Khanh', 'Tran', 'khanh.meeting3@example.com', 'N/A',
            'Vortex', 'VP Sales', 'N/A', 'Da Nang', 'Vietnam', 'N/A', 'N/A',
            '{}'::jsonb, '{}'::jsonb, '{}'::jsonb, '{}'::jsonb, '{"fit": 93, "engagement": 78, "priority": 88}'::jsonb,
            false, '{}'::jsonb
        )
        RETURNING id
    )
    SELECT array_agg(id) INTO v_contact_ids FROM inserted;

    INSERT INTO contact_segment_scores (
        id, contact_id, segmentation_id, fit_score, score_breakdown, score_type, status, passes_filters, enrolled_in_campaign
    )
    SELECT
        gen_random_uuid(), contact_id, v_segment_id, 95, '{}'::jsonb, 'manual', 'qualified', true, false
    FROM unnest(v_contact_ids) AS contact_id;

    RAISE NOTICE 'Seeded user_id=%', v_user_id;
    RAISE NOTICE 'Seeded organization_id=%', v_org_id;
    RAISE NOTICE 'Seeded segmentation_id=%', v_segment_id;
    RAISE NOTICE 'Seeded contact_ids=%', v_contact_ids;
END $$;
