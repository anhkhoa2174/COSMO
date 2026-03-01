import { Button } from "@/components/ui/button";
import { Dialog, DialogClose, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import Image from "next/image";
import { useEffect, useState } from "react";

export default function WelcomeDialog({
  open,
  onExplore,
  title = "Welcome to Cosmo Agents",
  description = "Let's get started with your AI Agents!",
  imageSrc = "/landing-page/header.png",
  textButton = "Explore",
}: {
  open: boolean;
  onExplore: () => void;
  title?: string | React.ReactNode;
  description?: string | React.ReactNode;
  imageSrc?: string;
  textButton?: string;
}) {
  const [isTourOpen, setIsTourOpen] = useState(open);

  const handleExplore = () => {
    setIsTourOpen(false);
    onExplore();
  };

  const handleOnOpenChange = (open: boolean) => {
    setIsTourOpen(open);
  };

  useEffect(() => {
    setIsTourOpen(open);
  }, [open]);

  return (
    <Dialog open={isTourOpen} onOpenChange={handleOnOpenChange}>
      <DialogContent
        onEscapeKeyDown={(e) => {
          e.preventDefault();
        }}
        onInteractOutside={(e) => {
          e.preventDefault();
        }}
        isCloseButton={false}
      >
        <DialogHeader className="flex flex-col items-center">
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription className="flex flex-col items-center w-full h-full">
            {imageSrc && (
              <Image src={imageSrc} alt="Cosmo Agents" width={500} height={500} />
            )}
            {description}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter className="flex justify-center">
          <DialogClose asChild>
            <Button onClick={handleExplore} className='w-60 mx-auto' variant="default">{textButton}</Button>
          </DialogClose>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
