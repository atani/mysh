import { Composition } from "remotion";
import { MyshDemo } from "./MyshDemo";

export const Root: React.FC = () => {
  return (
    <Composition
      id="MyshDemo"
      component={MyshDemo}
      durationInFrames={1500}
      fps={30}
      width={800}
      height={500}
    />
  );
};
