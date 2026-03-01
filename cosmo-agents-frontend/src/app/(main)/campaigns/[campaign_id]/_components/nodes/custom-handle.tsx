import React from 'react';
import { Handle, useHandleConnections } from '@xyflow/react';

const CustomHandle = (props: any) => {
  const connections = useHandleConnections({
    type: props.type,
  });

  return <Handle {...props} isConnectable={connections.length < props.connectioncount} />;
};

export default CustomHandle;
