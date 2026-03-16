import React from "react";
import { Composition } from "remotion";
import { NativeDriverDemo } from "./NativeDriverDemo";

export const Root: React.FC = () => {
  return (
    <Composition
      id="NativeDriverDemo"
      component={NativeDriverDemo}
      durationInFrames={900}
      fps={30}
      width={800}
      height={500}
    />
  );
};
