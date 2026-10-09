import React from 'react';
import { Card, Group, Stack, Text } from '@mantine/core';
import { Position, type NodeProps } from '@xyflow/react';
import { Braces, Forward } from 'lucide-react';

import classes from '@/styles/Campaign.module.css';
import { IconSparkles } from '@/assets/icons';
import CustomHandle from './custom-handle';
import { type ActionNode } from './index';

export function ActionNode({ data }: NodeProps<ActionNode>) {
  const items = [
    {
      icon: <IconSparkles width={20} height={20} />,
      label: 'Let AI reply',
    },
    {
      icon: <Forward width={20} height={20} />,
      label: 'Draft an email',
    },
    {
      icon: <Braces width={20} height={20} />,
      label: 'Assign to a person',
    },
  ];

  return (
    <>
      <CustomHandle
        id="node4-left"
        type="target"
        position={Position.Left}
        connectioncount={1}
      />
      <Card miw={256} p="sm" shadow="0px 4px 15px 0px #0000001A">
        <Text fw={600} mb="xs">
          Add Follow-up Action
        </Text>
        <Stack gap="xs">
          {items.map((action, index) => (
            <Group
              key={`action-${index}`}
              className={classes.listContact}
              onClick={data.actions[index]}
            >
              <Group>
                {action.icon}
                {action.label}
              </Group>
            </Group>
          ))}
        </Stack>
      </Card>
    </>
  );
}
